package postgres

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestPGWireClientAndEnterpriseCatalog(t *testing.T) {
	cat := NewEnterpriseCatalog()
	rows, err := cat.ExecuteSQLQuery("SELECT 1 AS alive;")
	if err != nil {
		t.Fatalf("ExecuteSQLQuery error: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected non-empty result from ExecuteSQLQuery")
	}
}

func writePGMsg(w io.Writer, typ byte, payload []byte) {
	var buf bytes.Buffer
	buf.WriteByte(typ)
	_ = binary.Write(&buf, binary.BigEndian, uint32(4+len(payload)))
	buf.Write(payload)
	_, _ = w.Write(buf.Bytes())
}

func readStartup(conn net.Conn) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:])) - 4
	if n > 0 {
		buf := make([]byte, n)
		_, _ = io.ReadFull(conn, buf)
	}
}

func startMockPGServer(t *testing.T, authMode string) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				readStartup(c)
				switch authMode {
				case "trust":
					writePGMsg(c, 'R', []byte{0, 0, 0, 0})
					writePGMsg(c, 'S', []byte("server_version\x0016.0\x00"))
					writePGMsg(c, 'Z', []byte{'I'})
				case "cleartext":
					writePGMsg(c, 'R', []byte{0, 0, 0, 3})
					_, _, _ = readPGMessage(c)
					writePGMsg(c, 'R', []byte{0, 0, 0, 0})
					writePGMsg(c, 'Z', []byte{'I'})
				case "md5":
					writePGMsg(c, 'R', []byte{0, 0, 0, 5, 1, 2, 3, 4})
					_, _, _ = readPGMessage(c)
					writePGMsg(c, 'R', []byte{0, 0, 0, 0})
					writePGMsg(c, 'Z', []byte{'I'})
				case "scram":
					writePGMsg(c, 'R', []byte{0, 0, 0, 10})
					_, initPayload, err := readPGMessage(c)
					if err != nil {
						return
					}
					// Extract client nonce r=...
					sInit := string(initPayload)
					idx := strings.Index(sInit, "r=")
					cNonce := "nonce"
					if idx >= 0 {
						cNonce = sInit[idx+2:]
					}
					saltB64 := base64.StdEncoding.EncodeToString([]byte("saltsalt"))
					srvFirst := "r=" + cNonce + "srv,s=" + saltB64 + ",i=2"
					var r11 bytes.Buffer
					_ = binary.Write(&r11, binary.BigEndian, uint32(11))
					r11.WriteString(srvFirst)
					writePGMsg(c, 'R', r11.Bytes())
					_, _, _ = readPGMessage(c)
					writePGMsg(c, 'R', []byte{0, 0, 0, 12})
					writePGMsg(c, 'Z', []byte{'I'})
				case "error":
					writePGMsg(c, 'E', []byte("SERROR\x00Mauth failed\x00\x00"))
					return
				case "unsupported_auth":
					writePGMsg(c, 'R', []byte{0, 0, 0, 99})
					return
				case "short_auth":
					writePGMsg(c, 'R', []byte{0, 0})
					return
				case "short_md5":
					writePGMsg(c, 'R', []byte{0, 0, 0, 5, 1})
					return
				case "scram_err":
					writePGMsg(c, 'R', []byte{0, 0, 0, 10})
					_, _, _ = readPGMessage(c)
					writePGMsg(c, 'E', []byte("SERROR\x00Mscram failed\x00\x00"))
					return
				}

				// Read SimpleQuery 'Q'
				_, _, err := readPGMessage(c)
				if err != nil {
					return
				}
				if authMode == "scram" {
					// Return query error response to cover 'E' in Query loop
					writePGMsg(c, 'E', []byte("SERROR\x00C42P01\x00Mrelation not found\x00\x00"))
					writePGMsg(c, 'Z', []byte{'I'})
					return
				}

				// Send RowDescription ('T') with 3 columns: id, content, null_col
				cols := []string{"id", "content", "null_col"}
				var tBuf bytes.Buffer
				_ = binary.Write(&tBuf, binary.BigEndian, uint16(len(cols)))
				for _, col := range cols {
					tBuf.WriteString(col)
					tBuf.WriteByte(0)
					tBuf.Write(make([]byte, 18))
				}
				writePGMsg(c, 'T', tBuf.Bytes())

				// Send DataRow ('D')
				var dBuf bytes.Buffer
				_ = binary.Write(&dBuf, binary.BigEndian, uint16(3))
				v1 := []byte("id-100")
				_ = binary.Write(&dBuf, binary.BigEndian, int32(len(v1)))
				dBuf.Write(v1)
				v2 := []byte("hello pgwire")
				_ = binary.Write(&dBuf, binary.BigEndian, int32(len(v2)))
				dBuf.Write(v2)
				_ = binary.Write(&dBuf, binary.BigEndian, int32(-1)) // NULL
				writePGMsg(c, 'D', dBuf.Bytes())

				writePGMsg(c, 'C', []byte("SELECT 1\x00"))
				writePGMsg(c, 'Z', []byte{'I'})
				_, _, _ = readPGMessage(c) // Terminate 'X'
			}(conn)
		}
	}()
	return ln.Addr().String()
}

func TestPostgresFullCoverageWithMockWireServers(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes")
	}

	// EnterpriseCatalog branches
	cat := NewEnterpriseCatalog()
	ten := cat.UpsertTenant(TenantRecord{Name: "Acme"})
	if ten.ID == "" {
		t.Fatal("expected generated tenant ID")
	}
	proj := cat.UpsertProject("proj-1", nil)
	if proj["project_id"] != "proj-1" || len(cat.ListProjects()) < 2 {
		t.Fatal("UpsertProject / ListProjects failed")
	}
	node := cat.RegisterComputeNode(ComputeNodeRecord{Hostname: "node4"})
	if node.NodeID == "" || node.Status != "ONLINE" {
		t.Fatal("RegisterComputeNode defaults failed")
	}
	task := cat.CreateTask("node1-lumina", "eval", nil)
	if _, ok := cat.GetTask(task.TaskID); !ok {
		t.Fatal("GetTask failed")
	}
	if len(cat.ListPendingTasks("node1-lumina")) != 1 || len(cat.ListPendingTasks("other-node")) != 0 {
		t.Fatal("ListPendingTasks filter failed")
	}
	_, _ = cat.CompleteTask("nonexistent-task", map[string]interface{}{"ok": true})

	// NewPGWireClient defaults & NewPGWireClientFromEnv
	defClient := NewPGWireClient("", "", "", "")
	if defClient.Addr != "127.0.0.1:5432" || defClient.User != "rae" || defClient.Database != "rae_memory" {
		t.Fatal("NewPGWireClient defaults failed")
	}
	envClient := NewPGWireClientFromEnv()
	if envClient.Database != "rae" {
		t.Fatal("NewPGWireClientFromEnv failed")
	}

	// Trust server: Ping, UpsertMemoryRow, DeleteMemoryRow, LoadAllMemories
	trustAddr := startMockPGServer(t, "trust")
	tc := NewPGWireClient(trustAddr, "rae", "pwd", "rae")
	if err := tc.Ping(); err != nil {
		t.Fatalf("Ping on trust server failed: %v", err)
	}
	if err := tc.UpsertMemoryRow(nil); err != nil {
		t.Fatalf("UpsertMemoryRow(nil) failed: %v", err)
	}
	if err := tc.UpsertMemoryRow(&models.MemoryItem{Content: " O'Reilly's book "}); err != nil {
		t.Fatalf("UpsertMemoryRow failed: %v", err)
	}
	if err := tc.DeleteMemoryRow("id-100"); err != nil {
		t.Fatalf("DeleteMemoryRow failed: %v", err)
	}
	items, err := tc.LoadAllMemories(0)
	if err != nil || len(items) != 1 || items[0].ID != "id-100" {
		t.Fatalf("LoadAllMemories failed: %+v, %v", items, err)
	}

	// Cleartext auth server
	clearAddr := startMockPGServer(t, "cleartext")
	cc := NewPGWireClient(clearAddr, "rae", "pwd", "rae")
	if err := cc.Ping(); err != nil {
		t.Fatalf("Ping on cleartext server failed: %v", err)
	}

	// MD5 auth server
	md5Addr := startMockPGServer(t, "md5")
	mc := NewPGWireClient(md5Addr, "rae", "pwd", "rae")
	if err := mc.Ping(); err != nil {
		t.Fatalf("Ping on md5 server failed: %v", err)
	}

	// SCRAM-SHA-256 auth + Query ErrorResponse
	scramAddr := startMockPGServer(t, "scram")
	sc := NewPGWireClient(scramAddr, "rae", "pwd", "rae")
	if _, err := sc.Query("SELECT * FROM missing"); err == nil {
		t.Fatal("expected query error from scram mock server")
	}

	// Auth ErrorResponse
	errAddr := startMockPGServer(t, "error")
	ec := NewPGWireClient(errAddr, "rae", "pwd", "rae")
	if err := ec.Ping(); err == nil {
		t.Fatal("expected auth error from error mock server")
	}

	// Direct readPGMessage invalid length test
	if _, _, err := readPGMessage(bytes.NewReader([]byte{'R', 0, 0, 0, 2})); err == nil {
		t.Fatal("expected error for msgLen < 4")
	}
	for _, mode := range []string{"unsupported_auth", "short_auth", "short_md5", "scram_err"} {
		addr := startMockPGServer(t, mode)
		_ = NewPGWireClient(addr, "rae", "pwd", "rae").Ping()
	}
}

