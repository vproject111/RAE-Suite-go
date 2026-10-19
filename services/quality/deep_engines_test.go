package quality

import (
	"testing"
)

func TestCatalog198DTPValidatorCalculatorAndGoldenSet(t *testing.T) {
	// 1. Catalog 198 + 12 AI/SaaS topics
	cat := BuildCatalog198Summary()
	if cat["engineering_topics"].(int) != 198 || cat["ai_saas_topics"].(int) != 12 || cat["total_catalog_topics"].(int) != 210 {
		t.Fatalf("expected 198 + 12 = 210 topics in catalog, got %+v", cat)
	}

	// 2. PDF DTP Validator: valid vs missing bleed & TAC > 300%
	validDoc := PDFDocumentSpec{
		DocumentID: "doc-ok",
		TrimBox:    BoxDimensions{Width: 90, Height: 50},
		BleedBox:   BoxDimensions{Width: 94, Height: 54},
		MinBleedMM: 2.0,
		DPIMin:     300,
		TextElements: []DTPTextElement{
			{FontName: "Inter", FontSizePt: 10, IsEmbedded: true, Color: CMYKColor{C: 0, M: 0, Y: 0, K: 100}},
		},
		SpotLayers: []DTPSpotLayer{
			{Name: "CutContour", Overprint: true},
		},
	}
	resOK := ValidatePDFDocument(validDoc)
	if !resOK.Passed || !resOK.HasValidBleed || resOK.BleedXMM != 2.0 {
		t.Fatalf("expected valid DTP document to pass, got %+v", resOK)
	}

	badDoc := PDFDocumentSpec{
		DocumentID: "doc-bad",
		TrimBox:    BoxDimensions{Width: 90, Height: 50},
		BleedBox:   BoxDimensions{Width: 90, Height: 50}, // missing bleed
		MinBleedMM: 2.0,
		DPIMin:     150, // low DPI
		TextElements: []DTPTextElement{
			{FontName: "UnembeddedFont", FontSizePt: 8, IsEmbedded: false, Color: CMYKColor{C: 90, M: 90, Y: 90, K: 90}}, // TAC 360% > 300%
		},
		SpotLayers: []DTPSpotLayer{
			{Name: "CutContour", Overprint: false},
		},
	}
	resBad := ValidatePDFDocument(badDoc)
	if resBad.Passed || resBad.HasValidBleed || resBad.CMYKCompliant || resBad.FontsEmbedded || len(resBad.Issues) < 4 {
		t.Fatalf("expected bad DTP document to fail with >=4 issues, got %+v", resBad)
	}

	// 3. Calculator Verifier: valid monotonic vs price inversion
	calcOK := VerifyCalculatorTiers([]PriceTier{
		{Volume: 100, TotalPrice: 50.0},
		{Volume: 500, TotalPrice: 150.0},
		{Volume: 1000, TotalPrice: 250.0},
	}, 50.0, 90.0, 2.0)
	if !calcOK.IsValid {
		t.Fatalf("expected monotonic pricing to pass, got %+v", calcOK)
	}

	calcBad := VerifyCalculatorTiers([]PriceTier{
		{Volume: 100, TotalPrice: 100.0},
		{Volume: 500, TotalPrice: 80.0}, // total price inversion!
	}, 100.0, 90.0, 2.0)
	if calcBad.IsValid || len(calcBad.Violations) < 2 {
		t.Fatalf("expected price inversion to be caught, got %+v", calcBad)
	}

	// 4. Risk Coverage & Golden Set
	rc := EvaluateRiskCoverageMatrix()
	if rc["verified"] != true {
		t.Fatalf("expected risk coverage matrix verified=true, got %+v", rc)
	}
	gs := RunCanonicalGoldenSetAudit()
	if gs["golden_set_passed"] != true || gs["passed_documents"].(int) != 5 {
		t.Fatalf("expected all 5 canonical golden set documents to pass, got %+v", gs)
	}
}

func TestBinaryPDFAndMultiLangAST(t *testing.T) {
	// 210x297mm is 595.28x841.89pt; 216x303mm is 612.28x858.90pt
	rawPDF := "%PDF-1.7\n1 0 obj << /Type /Page /TrimBox [0 0 595.28 841.89] /BleedBox [0 0 612.28 858.90] /ColorSpace /DeviceCMYK /Font << /F1 << /FontFile 2 0 R >> >> >> endobj\n%%EOF"
	pdfRes := ParseBinaryPDFAndValidate(rawPDF)
	if pdfRes["passed"] != true || pdfRes["pdf_header_detected"] != true || pdfRes["cmyk_colorspace"] != true {
		t.Fatalf("expected binary PDF stream to pass DTP audit, got %+v", pdfRes)
	}

	pyCode := "class Calculator:\n    def compute(self, x):\n        if x > 0:\n            return x * 2\n        return 0\n"
	pyRep := AnalyzeMultiLangAST("python", pyCode)
	if pyRep.FunctionsCount != 1 || pyRep.ClassesOrTypesCount != 1 || !pyRep.PassedQualityGate {
		t.Fatalf("expected Python AST report to pass with 1 func and 1 class, got %+v", pyRep)
	}

	goCode := "package main\ntype S struct{}\nfunc Add(a, b int) int {\n\tif a > 0 {\n\t\treturn a + b\n\t}\n\treturn b\n}\n"
	goRep := AnalyzeMultiLangAST("go", goCode)
	if goRep.FunctionsCount != 1 || goRep.ClassesOrTypesCount != 1 || !goRep.PassedQualityGate {
		t.Fatalf("expected Go AST report to pass with 1 func and 1 type, got %+v", goRep)
	}
}
