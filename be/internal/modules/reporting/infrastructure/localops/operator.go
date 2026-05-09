package localops

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	reportingdomain "be_ads_project/internal/modules/reporting/domain"
)

type Operator struct {
	rootDir string
}

func NewOperator(rootDir string) *Operator {
	return &Operator{rootDir: rootDir}
}

func (o *Operator) Start(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "start_local_stack", `
./scripts/dev/dev_base_stack_up.sh
START_SKIP_BI_API=1 ./scripts/ops/start.sh collector-worker transformer-worker control-plane
./scripts/ops/status.sh
`)
}

func (o *Operator) Stop(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.stopWorkers(ctx, "stop_local_stack")
}

func (o *Operator) Restart(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "restart_local_stack", `
source ./scripts/ops/common.sh
services=("control-plane")
while read -r name; do
  [[ -z "${name}" ]] && continue
  case "${name}" in
    collector-worker*|transformer-worker*)
      services+=("${name}")
      ;;
  esac
done < <(list_managed_service_names "./run")
./scripts/ops/stop.sh "${services[@]}"
./scripts/dev/dev_base_stack_up.sh
START_SKIP_BI_API=1 ./scripts/ops/start.sh collector-worker transformer-worker control-plane
./scripts/ops/status.sh
`)
}

func (o *Operator) Verify(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.run(ctx, "verify_local_stack", "scripts/verify/verify_local_stack.sh")
}

func (o *Operator) StartInfra(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "start_local_infra", `
./scripts/dev/dev_base_stack_up.sh
./scripts/ops/status.sh
`)
}

func (o *Operator) StopInfra(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "stop_local_infra", `
./scripts/dev/dev_base_stack_down.sh
./scripts/ops/status.sh
`)
}

func (o *Operator) StartWorkers(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "start_local_workers", `
START_SKIP_BI_API=1 ./scripts/ops/start.sh collector-worker transformer-worker control-plane
./scripts/ops/status.sh
`)
}

func (o *Operator) StopWorkers(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.stopWorkers(ctx, "stop_local_workers")
}

func (o *Operator) RestartCollector(ctx context.Context) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, "restart_collector_worker", `
./scripts/ops/stop.sh collector-worker
START_SKIP_BI_API=1 ./scripts/ops/start.sh collector-worker
./scripts/ops/status.sh
`)
}

func (o *Operator) AddWorker(ctx context.Context, role string) (*reportingdomain.LocalCommandResult, error) {
	baseName, pkg, err := workerScriptConfig(role)
	if err != nil {
		return nil, err
	}
	return o.runBash(ctx, "add_"+role+"_worker", fmt.Sprintf(`
source ./scripts/ops/common.sh
run_dir="${PWD}/run"
log_dir="${PWD}/logs"
boot_log="${log_dir}/startup.log"
base_name="%s"
pkg="%s"
mkdir -p "${run_dir}" "${log_dir}"
next_index=2
while [[ -f "${run_dir}/${base_name}-${next_index}.pid" || -x "${run_dir}/${base_name}-${next_index}" ]]; do
  next_index=$((next_index + 1))
done
name="${base_name}-${next_index}"
bin_path="${run_dir}/${name}"
pid_file="${run_dir}/${name}.pid"
stdout_log="${log_dir}/${name}.stdout.log"
echo "[build] ${name}" >>"${boot_log}"
go build -o "${bin_path}" "${pkg}" >>"${boot_log}" 2>&1
BE_WORKER_ID="${name}" nohup "${bin_path}" >>"${stdout_log}" 2>&1 < /dev/null &
pid=$!
write_pid_file "${pid_file}" "${pid}"
sleep 1
resolved_pid="$(resolve_service_pid "${pid_file}" "${bin_path}" 2>/dev/null || true)"
if [[ -z "${resolved_pid}" ]]; then
  echo "${name} failed to start, check ${stdout_log} and ${boot_log}"
  rm -f "${pid_file}"
  exit 1
fi
write_pid_file "${pid_file}" "${resolved_pid}"
echo "${name} started, pid=${resolved_pid}"
./scripts/ops/status.sh
`, baseName, pkg))
}

func (o *Operator) RemoveWorker(ctx context.Context, role string) (*reportingdomain.LocalCommandResult, error) {
	baseName, _, err := workerScriptConfig(role)
	if err != nil {
		return nil, err
	}
	return o.runBash(ctx, "remove_"+role+"_worker", fmt.Sprintf(`
run_dir="${PWD}/run"
base_name="%s"
target=""
max_index=1
if [[ -d "${run_dir}" ]]; then
  for pid_file in "${run_dir}/${base_name}-"*.pid; do
    [[ -e "${pid_file}" ]] || continue
    name="$(basename "${pid_file}" .pid)"
    index="${name##${base_name}-}"
    if [[ "${index}" =~ ^[0-9]+$ ]] && (( index > max_index )); then
      max_index="${index}"
      target="${name}"
    fi
  done
fi
if [[ -z "${target}" ]]; then
  echo "no extra %s worker to remove"
  ./scripts/ops/status.sh
  exit 0
fi
./scripts/ops/stop.sh "${target}"
rm -f "${run_dir}/${target}"
echo "${target} removed"
./scripts/ops/status.sh
`, baseName, role))
}

func (o *Operator) Status(ctx context.Context) (*reportingdomain.LocalStackStatus, error) {
	status := &reportingdomain.LocalStackStatus{
		Enabled:   true,
		UpdatedAt: time.Now().UTC(),
		Services:  make([]reportingdomain.LocalProcessState, 0),
		Workers:   make([]reportingdomain.LocalWorkerGroupState, 0),
		Infra:     make([]reportingdomain.LocalProcessState, 0),
		Ports:     make([]reportingdomain.LocalPortState, 0),
		Logs:      make([]reportingdomain.LocalLogState, 0),
	}
	status.Services = append(status.Services, o.collectServiceStates()...)
	status.Workers = append(status.Workers, buildWorkerGroups(status.Services)...)
	status.Infra = append(status.Infra, o.collectInfraStates(ctx)...)
	status.Ports = append(status.Ports, o.collectPortStates()...)
	status.Logs = append(status.Logs, o.collectLogStates()...)
	status.Output = o.buildSummary(status)
	return status, nil
}

func (o *Operator) run(ctx context.Context, action string, relativeScript string, args ...string) (*reportingdomain.LocalCommandResult, error) {
	if strings.TrimSpace(o.rootDir) == "" {
		return nil, fmt.Errorf("local ops root dir is empty")
	}
	scriptPath := filepath.Join(o.rootDir, relativeScript)
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.CommandContext(runCtx, "bash", cmdArgs...)
	cmd.Dir = o.rootDir
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	startedAt := time.Now().UTC()
	err := cmd.Run()
	finishedAt := time.Now().UTC()
	output := strings.TrimSpace(combined.String())

	result := &reportingdomain.LocalCommandResult{
		Action:     action,
		Success:    err == nil,
		Output:     output,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
	if err != nil {
		result.Error = err.Error()
		if output == "" {
			output = err.Error()
			result.Output = output
		}
		return result, fmt.Errorf("%s failed: %w", action, err)
	}
	return result, nil
}

func (o *Operator) stopWorkers(ctx context.Context, action string) (*reportingdomain.LocalCommandResult, error) {
	return o.runBash(ctx, action, `
source ./scripts/ops/common.sh
services=("control-plane")
while read -r name; do
  [[ -z "${name}" ]] && continue
  case "${name}" in
    collector-worker*|transformer-worker*)
      services+=("${name}")
      ;;
  esac
done < <(list_managed_service_names "./run")
./scripts/ops/stop.sh "${services[@]}"
./scripts/ops/status.sh
`)
}

func (o *Operator) runBash(ctx context.Context, action string, script string) (*reportingdomain.LocalCommandResult, error) {
	if strings.TrimSpace(o.rootDir) == "" {
		return nil, fmt.Errorf("local ops root dir is empty")
	}
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "bash", "-lc", "set -euo pipefail\n"+script)
	cmd.Dir = o.rootDir
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	startedAt := time.Now().UTC()
	err := cmd.Run()
	finishedAt := time.Now().UTC()
	output := strings.TrimSpace(combined.String())

	result := &reportingdomain.LocalCommandResult{
		Action:     action,
		Success:    err == nil,
		Output:     output,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
	if err != nil {
		result.Error = err.Error()
		if output == "" {
			result.Output = err.Error()
		}
		return result, fmt.Errorf("%s failed: %w", action, err)
	}
	return result, nil
}

func workerScriptConfig(role string) (string, string, error) {
	switch role {
	case "collector":
		return "collector-worker", "./cmd/collector-worker", nil
	case "transformer":
		return "transformer-worker", "./cmd/transformer-worker", nil
	default:
		return "", "", fmt.Errorf("unknown worker role: %s", role)
	}
}

func (o *Operator) collectServiceStates() []reportingdomain.LocalProcessState {
	services := discoverManagedServices(filepath.Join(o.rootDir, "run"))
	if len(services) == 0 {
		services = []string{"bi-api", "control-plane", "collector-worker", "transformer-worker"}
	}
	items := make([]reportingdomain.LocalProcessState, 0, len(services))
	for _, name := range services {
		pidFile := filepath.Join(o.rootDir, "run", name+".pid")
		binPath := filepath.Join(o.rootDir, "run", name)
		pid, ok := resolveServicePID(pidFile, binPath)
		state := reportingdomain.LocalProcessState{Name: name}
		switch {
		case ok:
			state.State = "running"
			state.Detail = fmt.Sprintf("pid=%s", pid)
		case fileExists(pidFile):
			state.State = "stale_pid"
			state.Detail = "pid file exists but process is gone"
		default:
			state.State = "stopped"
			state.Detail = "not running"
		}
		items = append(items, state)
	}
	return items
}

func (o *Operator) collectInfraStates(ctx context.Context) []reportingdomain.LocalProcessState {
	containers := []string{"be-ads-raw-mysql", "be-ads-serving-mysql", "be-ads-clickhouse", "be-ads-nats"}
	items := make([]reportingdomain.LocalProcessState, 0, len(containers))
	for _, name := range containers {
		state := reportingdomain.LocalProcessState{Name: name}
		stdout, err := runCommand(ctx, o.rootDir, "docker", "inspect", "-f", "{{.State.Status}}", name)
		if err != nil {
			state.State = "missing"
			state.Detail = "container not found"
		} else {
			value := strings.TrimSpace(stdout)
			if value == "" {
				value = "unknown"
			}
			state.State = value
			state.Detail = "docker container"
		}
		items = append(items, state)
	}
	return items
}

func (o *Operator) collectPortStates() []reportingdomain.LocalPortState {
	checks := []struct {
		name string
		port int
	}{
		{name: "bi-api", port: 8080},
		{name: "nats-monitor", port: 8222},
		{name: "raw-mysql", port: 3307},
		{name: "serving-mysql", port: 3308},
		{name: "clickhouse-http", port: 8123},
		{name: "clickhouse-native", port: 9000},
		{name: "nats", port: 4222},
	}
	items := make([]reportingdomain.LocalPortState, 0, len(checks))
	for _, check := range checks {
		addr := fmt.Sprintf("127.0.0.1:%d", check.port)
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		state := reportingdomain.LocalPortState{
			Name: check.name,
			Port: check.port,
		}
		if err != nil {
			state.State = "closed"
			state.Detail = err.Error()
		} else {
			_ = conn.Close()
			state.State = "listening"
			state.Detail = addr
		}
		items = append(items, state)
	}
	return items
}

func (o *Operator) collectLogStates() []reportingdomain.LocalLogState {
	serviceNames := discoverManagedServices(filepath.Join(o.rootDir, "run"))
	if len(serviceNames) == 0 {
		serviceNames = []string{"bi-api", "control-plane", "collector-worker", "transformer-worker"}
	}
	items := make([]reportingdomain.LocalLogState, 0, len(serviceNames))
	for _, serviceName := range serviceNames {
		lines, err := tailLines(filepath.Join(o.rootDir, "logs", serviceName+".stdout.log"), 4)
		state := reportingdomain.LocalLogState{Name: serviceName}
		if err != nil {
			state.State = "missing"
			state.Lines = []string{"log file not found"}
		} else {
			state.State = classifyLogState(lines)
			state.Lines = lines
		}
		items = append(items, state)
	}
	return items
}

func (o *Operator) buildSummary(status *reportingdomain.LocalStackStatus) string {
	var builder strings.Builder
	for _, group := range status.Workers {
		builder.WriteString(fmt.Sprintf("workers %s running=%d total=%d\n", group.Role, group.RunningCount, group.TotalCount))
	}
	for _, item := range status.Services {
		builder.WriteString(fmt.Sprintf("service %s %s %s\n", item.Name, item.State, strings.TrimSpace(item.Detail)))
	}
	for _, item := range status.Infra {
		builder.WriteString(fmt.Sprintf("infra %s %s %s\n", item.Name, item.State, strings.TrimSpace(item.Detail)))
	}
	for _, item := range status.Ports {
		builder.WriteString(fmt.Sprintf("port %s %s %d %s\n", item.Name, item.State, item.Port, strings.TrimSpace(item.Detail)))
	}
	return strings.TrimSpace(builder.String())
}

func discoverManagedServices(runDir string) []string {
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".pid") {
			continue
		}
		names = append(names, strings.TrimSuffix(name, ".pid"))
	}
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)
	return names
}

func buildWorkerGroups(services []reportingdomain.LocalProcessState) []reportingdomain.LocalWorkerGroupState {
	groupMap := map[string][]reportingdomain.LocalProcessState{
		"collector":   {},
		"transformer": {},
	}
	for _, service := range services {
		switch {
		case strings.HasPrefix(service.Name, "collector-worker"):
			groupMap["collector"] = append(groupMap["collector"], service)
		case strings.HasPrefix(service.Name, "transformer-worker"):
			groupMap["transformer"] = append(groupMap["transformer"], service)
		}
	}
	roles := []string{"collector", "transformer"}
	items := make([]reportingdomain.LocalWorkerGroupState, 0, len(roles))
	for _, role := range roles {
		instances := groupMap[role]
		running := 0
		for _, item := range instances {
			if item.State == "running" {
				running++
			}
		}
		items = append(items, reportingdomain.LocalWorkerGroupState{
			Role:         role,
			RunningCount: running,
			TotalCount:   len(instances),
			Instances:    instances,
		})
	}
	return items
}

func resolveServicePID(pidFile string, binPath string) (string, bool) {
	if data, err := os.ReadFile(pidFile); err == nil {
		pid := strings.TrimSpace(string(data))
		if pid != "" && processAlive(pid) {
			return pid, true
		}
	}
	stdout, err := runCommand(context.Background(), "", "ps", "ax", "-o", "pid=,comm=,args=")
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(stdout, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		cmd := fields[1]
		base := filepath.Base(binPath)
		if cmd != binPath && cmd != "./"+base && cmd != base {
			continue
		}
		if processAlive(fields[0]) {
			return fields[0], true
		}
	}
	return "", false
}

func processAlive(pid string) bool {
	cmd := exec.Command("kill", "-0", pid)
	return cmd.Run() == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runCommand(ctx context.Context, dir string, name string, args ...string) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}
		if output == "" {
			output = err.Error()
		}
		return "", fmt.Errorf("%s", output)
	}
	return stdout.String(), nil
}

func tailLines(path string, limit int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := make([]string, 0, limit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) > limit {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return []string{"no recent log lines"}, nil
	}
	return lines, nil
}

func classifyLogState(lines []string) string {
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") || strings.Contains(lower, " failed") || strings.Contains(lower, "error") {
			return "warn"
		}
	}
	return "ok"
}
