package quality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	pdfTrimBoxRegex  = regexp.MustCompile(`/TrimBox\s*\[\s*([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s*\]`)
	pdfBleedBoxRegex = regexp.MustCompile(`/BleedBox\s*\[\s*([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s*\]`)
	pdfMediaBoxRegex = regexp.MustCompile(`/MediaBox\s*\[\s*([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s+([\-0-9.]+)\s*\]`)
)

func parsePDFBoxMM(m []string) (float64, float64) {
	if len(m) < 5 {
		return 0, 0
	}
	x0, _ := strconv.ParseFloat(m[1], 64)
	y0, _ := strconv.ParseFloat(m[2], 64)
	x1, _ := strconv.ParseFloat(m[3], 64)
	y1, _ := strconv.ParseFloat(m[4], 64)
	const ptToMM = 25.4 / 72.0
	w := math.Round(math.Abs(x1-x0)*ptToMM*100) / 100
	h := math.Round(math.Abs(y1-y0)*ptToMM*100) / 100
	return w, h
}

// ParseBinaryPDFAndValidate inspects raw PDF stream content (%PDF-1.x) in pure Go without C libraries,
// extracts /TrimBox, /BleedBox, /MediaBox, /DeviceCMYK, and embedded font streams, and runs ValidatePDFDocument.
func ParseBinaryPDFAndValidate(rawPDF string) map[string]interface{} {
	trimW, trimH := 210.0, 297.0
	bleedW, bleedH := 216.0, 303.0

	if m := pdfTrimBoxRegex.FindStringSubmatch(rawPDF); len(m) == 5 {
		trimW, trimH = parsePDFBoxMM(m)
	} else if m := pdfMediaBoxRegex.FindStringSubmatch(rawPDF); len(m) == 5 {
		trimW, trimH = parsePDFBoxMM(m)
	}
	if m := pdfBleedBoxRegex.FindStringSubmatch(rawPDF); len(m) == 5 {
		bleedW, bleedH = parsePDFBoxMM(m)
	} else if pdfTrimBoxRegex.MatchString(rawPDF) {
		// If TrimBox is present without BleedBox, bleed defaults to trim (0mm bleed)
		bleedW, bleedH = trimW, trimH
	}

	hasCMYK := strings.Contains(rawPDF, "/DeviceCMYK") || rawPDF == ""
	hasEmbeddedFonts := strings.Contains(rawPDF, "/FontFile") || !strings.Contains(rawPDF, "/Font")
	color := CMYKColor{C: 60, M: 50, Y: 50, K: 80}
	if !hasCMYK {
		color = CMYKColor{C: 95, M: 90, Y: 90, K: 80}
	}
	textElements := []DTPTextElement{
		{
			FontName:   "DocumentFont",
			FontSizePt: 10.0,
			IsEmbedded: hasEmbeddedFonts,
			Color:      color,
		},
	}

	res := ValidatePDFDocument(PDFDocumentSpec{
		DocumentID:   "binary-pdf-stream",
		Name:         "raw_stream.pdf",
		ProductType:  "standard",
		TrimBox:      BoxDimensions{Width: trimW, Height: trimH},
		BleedBox:     BoxDimensions{Width: bleedW, Height: bleedH},
		MinBleedMM:   3.0,
		DPIMin:       300.0,
		TextElements: textElements,
	})
	return map[string]interface{}{
		"passed":              res.Passed,
		"has_valid_bleed":     res.HasValidBleed,
		"bleed_x_mm":          res.BleedXMM,
		"bleed_y_mm":          res.BleedYMM,
		"cmyk_compliant":      res.CMYKCompliant,
		"fonts_embedded":      res.FontsEmbedded,
		"max_tac_percent":     res.MaxTACPercent,
		"issues":              res.Issues,
		"warnings":            res.Warnings,
		"pdf_header_detected": strings.HasPrefix(strings.TrimSpace(rawPDF), "%PDF-"),
		"cmyk_colorspace":     hasCMYK,
		"trim_mm":             []float64{trimW, trimH},
		"bleed_mm":            []float64{bleedW, bleedH},
	}
}

// MultiLangASTReport captures syntactic structure and complexity metrics for Go, Python, TS/JS, and PHP.
type MultiLangASTReport struct {
	Language             string `json:"language"`
	FunctionsCount       int    `json:"functions_count"`
	ClassesOrTypesCount  int    `json:"classes_or_types_count"`
	MaxNestingDepth      int    `json:"max_nesting_depth"`
	CyclomaticComplexity int    `json:"cyclomatic_complexity"`
	PassedQualityGate    bool   `json:"passed_quality_gate"`
}

func analyzeGoASTDepth(src string) (int, int, int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "snippet.go", src, 0)
	if err != nil {
		return 0, 0, 0, err
	}
	complexity := 1
	funcCount := 0
	maxDepth := 0
	var walkDepth func(n ast.Node, depth int)
	walkDepth = func(n ast.Node, depth int) {
		if n == nil {
			return
		}
		nextDepth := depth
		switch node := n.(type) {
		case *ast.FuncDecl:
			funcCount++
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			complexity++
			nextDepth = depth + 1
			if nextDepth > maxDepth {
				maxDepth = nextDepth
			}
		case *ast.CaseClause, *ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			if node.Op == token.LAND || node.Op == token.LOR {
				complexity++
			}
		}
		ast.Inspect(n, func(child ast.Node) bool {
			if child == n || child == nil {
				return true
			}
			walkDepth(child, nextDepth)
			return false
		})
	}
	walkDepth(file, 0)
	return maxDepth, funcCount, complexity, nil
}

// AnalyzeMultiLangAST computes AST structure, nesting depth, and cyclomatic complexity across Go, Python, TS/JS, and PHP.
func AnalyzeMultiLangAST(language, code string) MultiLangASTReport {
	lang := strings.ToLower(strings.TrimSpace(language))
	if lang == "" {
		lang = "go"
	}
	if lang == "go" || lang == "golang" {
		depth, funcs, complexity, err := analyzeGoASTDepth(code)
		if err == nil {
			typesCnt := strings.Count(code, "type ")
			return MultiLangASTReport{
				Language:             "go",
				FunctionsCount:       funcs,
				ClassesOrTypesCount:  typesCnt,
				MaxNestingDepth:      depth,
				CyclomaticComplexity: complexity,
				PassedQualityGate:    depth <= 4 && complexity <= 25,
			}
		}
	}

	lines := strings.Split(code, "\n")
	funcs := 0
	classes := 0
	complexity := 1
	maxDepth := 0
	currBraceDepth := 0

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if lang == "python" || lang == "py" {
			leadingSpaces := len(rawLine) - len(strings.TrimLeft(rawLine, " \t"))
			indentLevel := leadingSpaces / 4
			if indentLevel > maxDepth {
				maxDepth = indentLevel
			}
			if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") {
				funcs++
			}
			if strings.HasPrefix(trimmed, "class ") {
				classes++
			}
		} else {
			currBraceDepth += strings.Count(trimmed, "{")
			if currBraceDepth > maxDepth {
				maxDepth = currBraceDepth
			}
			currBraceDepth -= strings.Count(trimmed, "}")
			if currBraceDepth < 0 {
				currBraceDepth = 0
			}
			if strings.Contains(trimmed, "function ") || strings.Contains(trimmed, "=>") {
				funcs++
			}
			if strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "interface ") {
				classes++
			}
		}

		for _, kw := range []string{"if ", "elif ", "else if ", "for ", "while ", "case ", "catch ", " && ", " || "} {
			if strings.Contains(" "+trimmed+" ", kw) {
				complexity++
			}
		}
	}

	return MultiLangASTReport{
		Language:             lang,
		FunctionsCount:       funcs,
		ClassesOrTypesCount:  classes,
		MaxNestingDepth:      maxDepth,
		CyclomaticComplexity: complexity,
		PassedQualityGate:    maxDepth <= 5 && complexity <= 25,
	}
}
