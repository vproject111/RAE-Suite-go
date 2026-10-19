package postgres

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// PGWireClient implements a pure-Go (CGO_ENABLED=0) PostgreSQL v3 Frontend/Backend Wire Protocol
// TCP client supporting StartupMessage (v3.0, 196608), Trust/Cleartext/MD5/SCRAM-SHA-256 authentication,
// SimpleQuery ('Q'), RowDescription ('T'), DataRow ('D'), CommandComplete ('C'), and ReadyForQuery ('Z').
type PGWireClient struct {
	mu       sync.Mutex
	Addr     string
	User     string
	Password string
	Database string
	Timeout  time.Duration
}

// NewPGWireClient creates a PostgreSQL wire client for the given TCP address and credentials.
func NewPGWireClient(addr, user, password, database string) *PGWireClient {
	if addr == "" {
		addr = "127.0.0.1:5432"
	}
	if user == "" {
		user = "rae"
	}
	if database == "" {
		database = "rae_memory"
	}
	return &PGWireClient{
		Addr:     addr,
		User:     user,
		Password: password,
		Database: database,
		Timeout:  2 * time.Second,
	}
}

// Ping tests TCP and PostgreSQL v3 wire protocol connectivity with "SELECT 1".
func (c *PGWireClient) Ping() error {
	rows, err := c.Query("SELECT 1 AS alive;")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("pgwire: empty response for SELECT 1")
	}
	return nil
}

// Query executes a SQL query over the PostgreSQL v3 wire protocol and returns decoded rows.
func (c *PGWireClient) Query(sqlQuery string) ([]map[string]interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", c.Addr, c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("pgwire dial %s: %w", c.Addr, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(c.Timeout))

	if err := c.handshake(conn); err != nil {
		return nil, err
	}

	// Send SimpleQuery ('Q')
	qBytes := append([]byte(sqlQuery), 0)
	msgLen := uint32(4 + len(qBytes))
	var qBuf bytes.Buffer
	qBuf.WriteByte('Q')
	_ = binary.Write(&qBuf, binary.BigEndian, msgLen)
	qBuf.Write(qBytes)
	if _, err := conn.Write(qBuf.Bytes()); err != nil {
		return nil, fmt.Errorf("pgwire write query: %w", err)
	}

	var columns []string
	var rows []map[string]interface{}
	var queryErr error

	for {
		msgType, payload, err := readPGMessage(conn)
		if err != nil {
			return rows, err
		}
		switch msgType {
		case 'T': // RowDescription
			if len(payload) >= 2 {
				numCols := int(binary.BigEndian.Uint16(payload[0:2]))
				columns = make([]string, 0, numCols)
				offset := 2
				for i := 0; i < numCols && offset < len(payload); i++ {
					end := bytes.IndexByte(payload[offset:], 0)
					if end < 0 {
						break
					}
					colName := string(payload[offset : offset+end])
					columns = append(columns, colName)
					offset += end + 1 + 18 // 18 bytes of field metadata (table OID, attr num, type OID, size, mod, format)
				}
			}
		case 'D': // DataRow
			if len(payload) >= 2 {
				numCols := int(binary.BigEndian.Uint16(payload[0:2]))
				row := make(map[string]interface{}, numCols)
				offset := 2
				for i := 0; i < numCols && offset+4 <= len(payload); i++ {
					colLen := int(int32(binary.BigEndian.Uint32(payload[offset : offset+4])))
					offset += 4
					colName := fmt.Sprintf("col_%d", i)
					if i < len(columns) {
						colName = columns[i]
					}
					if colLen < 0 {
						row[colName] = nil
					} else if offset+colLen <= len(payload) {
						val := string(payload[offset : offset+colLen])
						row[colName] = val
						offset += colLen
					}
				}
				rows = append(rows, row)
			}
		case 'C': // CommandComplete
			// Continue until ReadyForQuery ('Z')
		case 'E': // ErrorResponse
			queryErr = fmt.Errorf("pgwire error: %s", parsePGError(payload))
		case 'Z': // ReadyForQuery
			// Send Terminate ('X')
			_, _ = conn.Write([]byte{'X', 0, 0, 0, 4})
			return rows, queryErr
		}
	}
}

func (c *PGWireClient) handshake(conn net.Conn) error {
	var body bytes.Buffer
	_ = binary.Write(&body, binary.BigEndian, uint32(196608)) // Protocol v3.0
	body.WriteString("user")
	body.WriteByte(0)
	body.WriteString(c.User)
	body.WriteByte(0)
	body.WriteString("database")
	body.WriteByte(0)
	body.WriteString(c.Database)
	body.WriteByte(0)
	body.WriteByte(0)

	var startup bytes.Buffer
	_ = binary.Write(&startup, binary.BigEndian, uint32(4+body.Len()))
	startup.Write(body.Bytes())

	if _, err := conn.Write(startup.Bytes()); err != nil {
		return fmt.Errorf("pgwire write startup: %w", err)
	}

	for {
		msgType, payload, err := readPGMessage(conn)
		if err != nil {
			return fmt.Errorf("pgwire startup read: %w", err)
		}
		switch msgType {
		case 'R': // Authentication
			if len(payload) < 4 {
				return fmt.Errorf("pgwire short auth payload")
			}
			authType := binary.BigEndian.Uint32(payload[0:4])
			switch authType {
			case 0: // AuthenticationOk
				// Wait for ReadyForQuery ('Z')
			case 3: // AuthenticationCleartextPassword
				if err := sendPasswordMessage(conn, c.Password); err != nil {
					return err
				}
			case 5: // AuthenticationMD5Password
				if len(payload) < 8 {
					return fmt.Errorf("pgwire short md5 salt")
				}
				salt := payload[4:8]
				h1 := md5.Sum([]byte(c.Password + c.User))
				h1Hex := hex.EncodeToString(h1[:])
				h2Input := append([]byte(h1Hex), salt...)
				h2 := md5.Sum(h2Input)
				md5Pwd := "md5" + hex.EncodeToString(h2[:])
				if err := sendPasswordMessage(conn, md5Pwd); err != nil {
					return err
				}
			case 10: // AuthenticationSASL (SCRAM-SHA-256)
				if err := c.handleSCRAMSHA256(conn); err != nil {
					return err
				}
			default:
				return fmt.Errorf("pgwire unsupported auth type: %d", authType)
			}
		case 'S', 'K': // ParameterStatus, BackendKeyData
			// Ignore and continue waiting for 'Z'
		case 'E':
			return fmt.Errorf("pgwire auth error: %s", parsePGError(payload))
		case 'Z': // ReadyForQuery
			return nil
		}
	}
}

func (c *PGWireClient) handleSCRAMSHA256(conn net.Conn) error {
	nonceBytes := make([]byte, 18)
	_, _ = rand.Read(nonceBytes)
	clientNonce := base64.StdEncoding.EncodeToString(nonceBytes)
	clientFirstBare := fmt.Sprintf("n=,r=%s", clientNonce)
	clientFirstMsg := "n,," + clientFirstBare

	var saslInitial bytes.Buffer
	saslInitial.WriteByte('p')
	mech := "SCRAM-SHA-256\x00"
	totalLen := 4 + len(mech) + 4 + len(clientFirstMsg)
	_ = binary.Write(&saslInitial, binary.BigEndian, uint32(totalLen))
	saslInitial.WriteString(mech)
	_ = binary.Write(&saslInitial, binary.BigEndian, uint32(len(clientFirstMsg)))
	saslInitial.WriteString(clientFirstMsg)
	if _, err := conn.Write(saslInitial.Bytes()); err != nil {
		return err
	}

	msgType, payload, err := readPGMessage(conn)
	if err != nil {
		return err
	}
	if msgType == 'E' {
		return fmt.Errorf("scram initial error: %s", parsePGError(payload))
	}
	if msgType != 'R' || len(payload) < 4 || binary.BigEndian.Uint32(payload[0:4]) != 11 {
		return fmt.Errorf("expected AuthenticationSASLContinue (11), got %c", msgType)
	}
	serverFirst := string(payload[4:])
	parts := strings.Split(serverFirst, ",")
	var serverNonce, saltB64 string
	iterations := 4096
	for _, p := range parts {
		if strings.HasPrefix(p, "r=") {
			serverNonce = strings.TrimPrefix(p, "r=")
		} else if strings.HasPrefix(p, "s=") {
			saltB64 = strings.TrimPrefix(p, "s=")
		} else if strings.HasPrefix(p, "i=") {
			_, _ = fmt.Sscanf(strings.TrimPrefix(p, "i="), "%d", &iterations)
		}
	}
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return err
	}

	saltedPassword := pbkdf2SHA256([]byte(c.Password), salt, iterations)
	clientKey := hmacSHA256(saltedPassword, []byte("Client Key"))
	storedKey := sha256.Sum256(clientKey)
	clientFinalWithoutProof := fmt.Sprintf("c=biws,r=%s", serverNonce)
	authMessage := fmt.Sprintf("%s,%s,%s", clientFirstBare, serverFirst, clientFinalWithoutProof)
	clientSignature := hmacSHA256(storedKey[:], []byte(authMessage))
	clientProof := make([]byte, len(clientKey))
	for i := range clientKey {
		clientProof[i] = clientKey[i] ^ clientSignature[i]
	}
	clientFinal := fmt.Sprintf("%s,p=%s", clientFinalWithoutProof, base64.StdEncoding.EncodeToString(clientProof))

	var saslResp bytes.Buffer
	saslResp.WriteByte('p')
	_ = binary.Write(&saslResp, binary.BigEndian, uint32(4+len(clientFinal)))
	saslResp.WriteString(clientFinal)
	if _, err := conn.Write(saslResp.Bytes()); err != nil {
		return err
	}

	msgType2, payload2, err := readPGMessage(conn)
	if err != nil {
		return err
	}
	if msgType2 == 'E' {
		return fmt.Errorf("scram final error: %s", parsePGError(payload2))
	}
	return nil
}

func pbkdf2SHA256(password, salt []byte, iter int) []byte {
	mac := hmac.New(sha256.New, password)
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	u := mac.Sum(nil)
	out := make([]byte, len(u))
	copy(out, u)
	for i := 2; i <= iter; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func sendPasswordMessage(conn net.Conn, pwd string) error {
	var buf bytes.Buffer
	buf.WriteByte('p')
	pBytes := append([]byte(pwd), 0)
	_ = binary.Write(&buf, binary.BigEndian, uint32(4+len(pBytes)))
	buf.Write(pBytes)
	_, err := conn.Write(buf.Bytes())
	return err
}

func readPGMessage(r io.Reader) (byte, []byte, error) {
	var header [5]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}
	msgType := header[0]
	msgLen := int(binary.BigEndian.Uint32(header[1:5]))
	if msgLen < 4 {
		return msgType, nil, fmt.Errorf("pgwire invalid message length %d", msgLen)
	}
	payload := make([]byte, msgLen-4)
	if len(payload) > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return msgType, nil, err
		}
	}
	return msgType, payload, nil
}

func parsePGError(payload []byte) string {
	var parts []string
	offset := 0
	for offset < len(payload) && payload[offset] != 0 {
		fieldType := payload[offset]
		offset++
		end := bytes.IndexByte(payload[offset:], 0)
		if end < 0 {
			break
		}
		val := string(payload[offset : offset+end])
		if fieldType == 'M' || fieldType == 'S' || fieldType == 'C' {
			parts = append(parts, val)
		}
		offset += end + 1
	}
	return strings.Join(parts, ": ")
}
