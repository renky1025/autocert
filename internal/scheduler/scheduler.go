package scheduler

import (
	"autocert/internal/logger"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// TaskScheduler 任务调度器接口
type TaskScheduler interface {
	Install(taskName, command, schedule string) error
	Remove(taskName string) error
	List() ([]Task, error)
	IsInstalled(taskName string) bool
}

// Task 定时任务信息
type Task struct {
	Name     string
	Command  string
	Schedule string
	Status   string
	LastRun  string
	NextRun  string
}

// NewScheduler 创建新的任务调度器
func NewScheduler() TaskScheduler {
	if runtime.GOOS == "windows" {
		return &WindowsScheduler{}
	} else {
		return &LinuxScheduler{}
	}
}

// WindowsScheduler Windows 任务计划程序
type WindowsScheduler struct{}

// Install 安装 Windows 定时任务
func (w *WindowsScheduler) Install(taskName, command, schedule string) error {
	logger.Info("安装 Windows 定时任务", "taskName", taskName)

	// 转换调度格式
	windowsSchedule, err := w.convertSchedule(schedule)
	if err != nil {
		return fmt.Errorf("转换调度格式失败: %w", err)
	}

	// 生成 XML 配置
	xmlContent, err := w.generateTaskXML(taskName, command, windowsSchedule)
	if err != nil {
		return fmt.Errorf("生成任务 XML 失败: %w", err)
	}

	// 创建临时 XML 文件
	tempFile := filepath.Join(os.TempDir(), taskName+".xml")
	if err := os.WriteFile(tempFile, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tempFile)

	// 使用 schtasks 创建任务
	cmd := exec.Command("schtasks", "/create", "/tn", taskName, "/xml", tempFile, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建 Windows 任务失败: %s", string(output))
	}

	logger.Info("Windows 定时任务安装成功", "taskName", taskName)
	return nil
}

// Remove 删除 Windows 定时任务
func (w *WindowsScheduler) Remove(taskName string) error {
	logger.Info("删除 Windows 定时任务", "taskName", taskName)

	cmd := exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除 Windows 任务失败: %s", string(output))
	}

	logger.Info("Windows 定时任务删除成功", "taskName", taskName)
	return nil
}

// List 列出 Windows 定时任务
func (w *WindowsScheduler) List() ([]Task, error) {
	cmd := exec.Command("schtasks", "/query", "/fo", "csv", "/v")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查询 Windows 任务失败: %w", err)
	}

	// 解析 CSV 输出
	tasks := []Task{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] { // 跳过标题行
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) >= 4 {
			task := Task{
				Name:    strings.Trim(fields[0], "\""),
				Status:  strings.Trim(fields[3], "\""),
				LastRun: strings.Trim(fields[4], "\""),
				NextRun: strings.Trim(fields[5], "\""),
			}
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// IsInstalled 检查 Windows 任务是否已安装
func (w *WindowsScheduler) IsInstalled(taskName string) bool {
	cmd := exec.Command("schtasks", "/query", "/tn", taskName)
	err := cmd.Run()
	return err == nil
}

// convertSchedule 转换调度格式
func (w *WindowsScheduler) convertSchedule(schedule string) (string, error) {
	// 这里可以实现 cron 格式到 Windows 调度格式的转换
	// 简化处理，直接返回每日执行
	return "DAILY", nil
}

// generateTaskXML 生成任务 XML 配置
func (w *WindowsScheduler) generateTaskXML(taskName, command, schedule string) (string, error) {
	tmpl := `<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Date>2024-01-01T00:00:00</Date>
    <Author>AutoCert</Author>
    <Description>{{.TaskName}} - AutoCert 自动证书续期任务</Description>
  </RegistrationInfo>
  <Triggers>
    <CalendarTrigger>
      <StartBoundary>2024-01-01T02:00:00</StartBoundary>
      <Enabled>true</Enabled>
      <ScheduleByDay>
        <DaysInterval>1</DaysInterval>
      </ScheduleByDay>
    </CalendarTrigger>
  </Triggers>
  <Principals>
    <Principal id="Author">
      <UserId>S-1-5-18</UserId>
      <RunLevel>HighestAvailable</RunLevel>
    </Principal>
  </Principals>
  <Settings>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>
    <AllowHardTerminate>true</AllowHardTerminate>
    <StartWhenAvailable>true</StartWhenAvailable>
    <RunOnlyIfNetworkAvailable>true</RunOnlyIfNetworkAvailable>
    <IdleSettings>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
    <AllowStartOnDemand>true</AllowStartOnDemand>
    <Enabled>true</Enabled>
    <Hidden>false</Hidden>
    <RunOnlyIfIdle>false</RunOnlyIfIdle>
    <WakeToRun>false</WakeToRun>
    <ExecutionTimeLimit>PT1H</ExecutionTimeLimit>
    <Priority>7</Priority>
  </Settings>
  <Actions Context="Author">
    <Exec>
      <Command>{{.Command}}</Command>
      <Arguments>renew</Arguments>
    </Exec>
  </Actions>
</Task>`

	t, err := template.New("task").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := struct {
		TaskName string
		Command  string
	}{
		TaskName: taskName,
		Command:  command,
	}

	var result strings.Builder
	if err := t.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// LinuxScheduler Linux Cron 调度器
type LinuxScheduler struct{}

// Install 安装 Linux 定时任务
func (l *LinuxScheduler) Install(taskName, command, schedule string) error {
	logger.Info("安装 Linux 定时任务", "taskName", taskName)

	// 检查是否支持 systemd timer
	if l.supportsSystemdTimer() {
		return l.installSystemdTimer(taskName, command, schedule)
	} else {
		return l.installCronJob(taskName, command, schedule)
	}
}

// Remove 删除 Linux 定时任务
func (l *LinuxScheduler) Remove(taskName string) error {
	logger.Info("删除 Linux 定时任务", "taskName", taskName)

	if l.supportsSystemdTimer() {
		return l.removeSystemdTimer(taskName)
	} else {
		return l.removeCronJob(taskName)
	}
}

// List 列出 Linux 定时任务
func (l *LinuxScheduler) List() ([]Task, error) {
	if l.supportsSystemdTimer() {
		return l.listSystemdTimers()
	} else {
		return l.listCronJobs()
	}
}

// IsInstalled 检查 Linux 任务是否已安装
func (l *LinuxScheduler) IsInstalled(taskName string) bool {
	if l.supportsSystemdTimer() {
		return l.isSystemdTimerInstalled(taskName)
	} else {
		return l.isCronJobInstalled(taskName)
	}
}

// supportsSystemdTimer 检查是否支持 systemd timer
func (l *LinuxScheduler) supportsSystemdTimer() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// installSystemdTimer 安装 systemd timer
func (l *LinuxScheduler) installSystemdTimer(taskName, command, schedule string) error {
	// 创建 service 文件
	serviceContent := fmt.Sprintf(`[Unit]
Description=%s - AutoCert Certificate Renewal
After=network.target

[Service]
Type=oneshot
ExecStart=%s renew
User=root
`, taskName, command)

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", taskName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("创建 service 文件失败: %w", err)
	}

	// 创建 timer 文件
	timerContent := fmt.Sprintf(`[Unit]
Description=%s Timer - AutoCert Certificate Renewal
Requires=%s.service

[Timer]
OnCalendar=daily
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
`, taskName, taskName)

	timerPath := fmt.Sprintf("/etc/systemd/system/%s.timer", taskName)
	if err := os.WriteFile(timerPath, []byte(timerContent), 0644); err != nil {
		return fmt.Errorf("创建 timer 文件失败: %w", err)
	}

	// 重载 systemd 配置
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("重载 systemd 配置失败: %w", err)
	}

	// 启用并启动 timer
	if err := exec.Command("systemctl", "enable", taskName+".timer").Run(); err != nil {
		return fmt.Errorf("启用 timer 失败: %w", err)
	}

	if err := exec.Command("systemctl", "start", taskName+".timer").Run(); err != nil {
		return fmt.Errorf("启动 timer 失败: %w", err)
	}

	logger.Info("systemd timer 安装成功", "taskName", taskName)
	return nil
}

// removeSystemdTimer 删除 systemd timer
func (l *LinuxScheduler) removeSystemdTimer(taskName string) error {
	// 停止并禁用 timer
	exec.Command("systemctl", "stop", taskName+".timer").Run()
	exec.Command("systemctl", "disable", taskName+".timer").Run()

	// 删除文件
	os.Remove(fmt.Sprintf("/etc/systemd/system/%s.service", taskName))
	os.Remove(fmt.Sprintf("/etc/systemd/system/%s.timer", taskName))

	// 重载 systemd 配置
	exec.Command("systemctl", "daemon-reload").Run()

	logger.Info("systemd timer 删除成功", "taskName", taskName)
	return nil
}

// listSystemdTimers 列出 systemd timers
func (l *LinuxScheduler) listSystemdTimers() ([]Task, error) {
	cmd := exec.Command("systemctl", "list-timers", "--all", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("查询 systemd timers 失败: %w", err)
	}

	tasks := []Task{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "autocert") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				task := Task{
					Name:    fields[4],
					NextRun: fields[0] + " " + fields[1],
					Status:  "enabled",
				}
				tasks = append(tasks, task)
			}
		}
	}

	return tasks, nil
}

// isSystemdTimerInstalled 检查 systemd timer 是否已安装
func (l *LinuxScheduler) isSystemdTimerInstalled(taskName string) bool {
	cmd := exec.Command("systemctl", "is-enabled", taskName+".timer")
	err := cmd.Run()
	return err == nil
}

// installCronJob 安装 cron 任务
func (l *LinuxScheduler) installCronJob(taskName, command, schedule string) error {
	// 获取当前 crontab
	cmd := exec.Command("crontab", "-l")
	currentCrontab, _ := cmd.Output()

	// 添加新任务
	cronEntry := fmt.Sprintf("%s %s renew # %s\n", schedule, command, taskName)
	newCrontab := string(currentCrontab) + cronEntry

	// 写入新的 crontab
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("安装 cron 任务失败: %w", err)
	}

	logger.Info("cron 任务安装成功", "taskName", taskName)
	return nil
}

// removeCronJob 删除 cron 任务
func (l *LinuxScheduler) removeCronJob(taskName string) error {
	// 获取当前 crontab
	cmd := exec.Command("crontab", "-l")
	currentCrontab, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取 crontab 失败: %w", err)
	}

	// 过滤掉指定任务
	lines := strings.Split(string(currentCrontab), "\n")
	var newLines []string

	for _, line := range lines {
		if !strings.Contains(line, "# "+taskName) {
			newLines = append(newLines, line)
		}
	}

	// 写入新的 crontab
	newCrontab := strings.Join(newLines, "\n")
	cmd = exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("删除 cron 任务失败: %w", err)
	}

	logger.Info("cron 任务删除成功", "taskName", taskName)
	return nil
}

// listCronJobs 列出 cron 任务
func (l *LinuxScheduler) listCronJobs() ([]Task, error) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return []Task{}, nil // 没有 crontab 不是错误
	}

	tasks := []Task{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				task := Task{
					Name:     "cron-task",
					Command:  strings.Join(fields[5:], " "),
					Schedule: strings.Join(fields[0:5], " "),
					Status:   "enabled",
				}
				tasks = append(tasks, task)
			}
		}
	}

	return tasks, nil
}

// isCronJobInstalled 检查 cron 任务是否已安装
func (l *LinuxScheduler) isCronJobInstalled(taskName string) bool {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "# "+taskName)
}
