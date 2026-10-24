package main

import (
	"errors"
	"net/http"
	"os"
	"testing"
)

func TestRaeCLICommands(t *testing.T) {
	origArgs := os.Args
	origListen := listenAndServe
	origExit := exitFunc
	defer func() {
		os.Args = origArgs
		listenAndServe = origListen
		exitFunc = origExit
	}()

	exited := 0
	exitFunc = func(code int) { exited = code }

	runBootstrapCLI([]string{"--project=test-proj"})
	runMeshCLI([]string{"status"})
	runMeshCLI([]string{"sync"})
	runTriadCLI([]string{"--cascade", "--text=Test cascade"})
	runTriadCLI([]string{"--tier=tier1_generate", "--text=Test tier1"})
	runRecordTaskCLI([]string{"--task=Unit test CLI", "--rationale=Verify CLI coverage", "--effect=100% green"})
	runRecordTaskCLI([]string{})
	if exited != 1 {
		t.Fatalf("expected exit 1 on missing record-task args")
	}
	runDoctorCLI()
	runBenchmarkCLI()
	runExportCLI([]string{"out.jsonl"})
	runImportCLI([]string{"in.jsonl"})
	runInitCLI([]string{"--profile=edge"})
	runStatusCLI()

	listenAndServe = func(addr string, handler http.Handler) error {
		return errors.New("test stop")
	}

	for _, sub := range [][]string{
		{"rae", "bootstrap"},
		{"rae", "mesh", "status"},
		{"rae", "triad", "--text=hi"},
		{"rae", "record-task", "--task=t", "--rationale=r", "--effect=e"},
		{"rae", "doctor"},
		{"rae", "benchmark"},
		{"rae", "export"},
		{"rae", "import"},
		{"rae", "init"},
		{"rae", "status"},
		{"rae", "-dry-run", "-all=false", "-modules=memory,crl"},
		{"rae", "-all=true", "-port-offset=10000"},
	} {
		os.Args = sub
		main()
	}
}
