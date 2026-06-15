package main

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestProxyEnvArgs_EmptyWhenNoProxyEnv(t *testing.T) {
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
		"http_proxy", "https_proxy", "all_proxy",
		"NO_PROXY", "no_proxy"} {
		t.Setenv(k, "")
	}

	cm := &ContainerManager{cfg: &Config{DataDir: t.TempDir()}}
	got := cm.proxyEnvArgs()
	if len(got) != 0 {
		t.Fatalf("expected no args when no proxy env is set, got %v", got)
	}
}

// readEnvFile parses a docker --env-file into a map. Fails the test if the
// path doesn't exist or the args aren't in --env-file form.
func readEnvFile(t *testing.T, args []string) map[string]string {
	t.Helper()
	if len(args) != 2 || args[0] != "--env-file" {
		t.Fatalf("expected [--env-file PATH], got %v", args)
	}
	data, err := os.ReadFile(args[1])
	if err != nil {
		t.Fatalf("read env-file %s: %v", args[1], err)
	}
	info, _ := os.Stat(args[1])
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("env-file perm = %o, want 0600", info.Mode().Perm())
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			t.Fatalf("malformed env-file line: %q", line)
		}
		out[line[:idx]] = line[idx+1:]
	}
	return out
}

func TestProxyEnvArgs_RewritesLoopbackToHostDockerInternal(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:7899")
	t.Setenv("HTTPS_PROXY", "http://localhost:7899")
	t.Setenv("NO_PROXY", "localhost,127.0.0.1,corp.lan")

	cm := &ContainerManager{cfg: &Config{DataDir: t.TempDir()}}
	got := cm.proxyEnvArgs()

	pairs := readEnvFile(t, got)

	if v := pairs["HTTP_PROXY"]; v != "http://host.docker.internal:7899" {
		t.Errorf("HTTP_PROXY not rewritten: got %q", v)
	}
	if v := pairs["HTTPS_PROXY"]; v != "http://host.docker.internal:7899" {
		t.Errorf("HTTPS_PROXY not rewritten: got %q", v)
	}
	if v := pairs["NO_PROXY"]; v != "localhost,127.0.0.1,corp.lan" {
		t.Errorf("NO_PROXY must pass through unchanged: got %q", v)
	}
}

func TestProxyEnvArgs_PassesThroughRemoteProxyUnchanged(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.corp.lan:3128")

	cm := &ContainerManager{cfg: &Config{DataDir: t.TempDir()}}
	got := cm.proxyEnvArgs()

	pairs := readEnvFile(t, got)
	if v := pairs["HTTPS_PROXY"]; v != "http://proxy.corp.lan:3128" {
		t.Errorf("remote proxy URL must not be rewritten: got %q", v)
	}
}

func TestProxyEnvArgs_HandlesLowercase(t *testing.T) {
	t.Setenv("http_proxy", "http://127.0.0.1:7899")

	cm := &ContainerManager{cfg: &Config{DataDir: t.TempDir()}}
	got := cm.proxyEnvArgs()

	pairs := readEnvFile(t, got)
	if v := pairs["http_proxy"]; v != "http://host.docker.internal:7899" {
		t.Errorf("lowercase http_proxy not rewritten: got %q", v)
	}
}

// Credentials in proxy URLs must land in the env-file, never on the docker
// CLI where docker inspect / ps would expose them.
func TestProxyEnvArgs_CredentialsInFileNotCLIArgs(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://user:hunter2@127.0.0.1:3128")

	cm := &ContainerManager{cfg: &Config{DataDir: t.TempDir()}}
	got := cm.proxyEnvArgs()

	for _, a := range got {
		if strings.Contains(a, "hunter2") {
			t.Errorf("credential leaked into docker arg: %q", a)
		}
	}
	pairs := readEnvFile(t, got)
	if !strings.Contains(pairs["HTTPS_PROXY"], "hunter2") {
		t.Errorf("credential missing from env-file: %q", pairs["HTTPS_PROXY"])
	}
	if !strings.Contains(pairs["HTTPS_PROXY"], "host.docker.internal") {
		t.Errorf("loopback not rewritten: %q", pairs["HTTPS_PROXY"])
	}
}

// L1 fix: a path/query containing the literal "127.0.0.1" must not be
// rewritten — only the host portion is.
func TestRewriteLoopbackHost_PathUnchanged(t *testing.T) {
	in := "http://proxy.corp:3128/redirect?u=http://127.0.0.1:9999/x"
	got := rewriteLoopbackHost(in)
	if got != in {
		t.Errorf("expected unchanged, got %q", got)
	}
	got2 := rewriteLoopbackHost("http://127.0.0.1:3128/redirect?u=http://127.0.0.1:9999/x")
	if !strings.HasPrefix(got2, "http://host.docker.internal:3128/") {
		t.Errorf("host not rewritten: %q", got2)
	}
	if strings.Contains(got2[30:], "host.docker.internal") {
		t.Errorf("path/query wrongly rewritten: %q", got2)
	}
}

func TestMain(m *testing.M) {
	for _, k := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
		"http_proxy", "https_proxy", "all_proxy",
		"NO_PROXY", "no_proxy"} {
		_ = os.Unsetenv(k)
	}
	os.Exit(m.Run())
}
