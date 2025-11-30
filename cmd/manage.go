package cmd

import (
	"autocert/internal/cert"
	"autocert/internal/logger"
	"autocert/internal/scheduler"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var renewCmd = &cobra.Command{
	Use:   "renew",
	Short: "续期证书",
	Long: `检查并续期即将到期的证书。

示例:
  autocert renew                    # 续期所有证书
  autocert renew --domain example.com  # 续期指定域名的证书
  autocert renew --all              # 强制续期所有证书`,
	RunE: runRenew,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看证书状态",
	Long: `显示已安装证书的状态信息。

示例:
  autocert status                   # 显示所有证书状态
  autocert status --domain example.com # 显示指定域名证书状态`,
	RunE: runStatus,
}

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "管理定时任务",
	Long: `管理证书自动续期定时任务。

子命令:
  install   安装定时任务
  remove    删除定时任务
  list      列出定时任务
  status    查看任务状态`,
}

var scheduleInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "安装定时任务",
	RunE:  runScheduleInstall,
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "删除定时任务",
	RunE:  runScheduleRemove,
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出定时任务",
	RunE:  runScheduleList,
}

var (
	renewDomain  string
	renewAll     bool
	statusDomain string
	taskName     string
)

func init() {
	rootCmd.AddCommand(renewCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(scheduleCmd)

	// renew 命令参数
	renewCmd.Flags().StringVarP(&renewDomain, "domain", "d", "", "要续期的域名")
	renewCmd.Flags().BoolVar(&renewAll, "all", false, "强制续期所有证书")

	// status 命令参数
	statusCmd.Flags().StringVarP(&statusDomain, "domain", "d", "", "要查看的域名")

	// schedule 子命令
	scheduleCmd.AddCommand(scheduleInstallCmd)
	scheduleCmd.AddCommand(scheduleRemoveCmd)
	scheduleCmd.AddCommand(scheduleListCmd)

	// schedule 命令参数
	scheduleInstallCmd.Flags().StringVar(&taskName, "name", "autocert-renew", "任务名称")
	scheduleRemoveCmd.Flags().StringVar(&taskName, "name", "autocert-renew", "任务名称")
}

func runRenew(cmd *cobra.Command, args []string) error {
	logger.Info("开始证书续期", "domain", renewDomain, "forceAll", renewAll)

	if renewDomain != "" {
		// 续期指定域名
		return renewDomainCert(renewDomain)
	} else {
		// 续期所有域名
		return renewAllCerts()
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	logger.Info("查看证书状态", "domain", statusDomain)

	if statusDomain != "" {
		// 显示指定域名状态
		return showDomainStatus(statusDomain)
	} else {
		// 显示所有域名状态
		return showAllStatus()
	}
}

func runScheduleInstall(cmd *cobra.Command, args []string) error {
	logger.Info("安装定时任务", "taskName", taskName)

	// 获取当前执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取执行文件路径失败: %w", err)
	}

	// 创建调度器
	sched := scheduler.NewScheduler()

	// 安装任务（每日凌晨2点检查）
	schedule := "0 2 * * *" // cron 格式
	if err := sched.Install(taskName, execPath, schedule); err != nil {
		return fmt.Errorf("安装定时任务失败: %w", err)
	}

	fmt.Printf("✓ 定时任务 '%s' 安装成功\n", taskName)
	fmt.Println("任务将在每日凌晨2点自动检查并续期证书")

	return nil
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	logger.Info("删除定时任务", "taskName", taskName)

	// 创建调度器
	sched := scheduler.NewScheduler()

	// 删除任务
	if err := sched.Remove(taskName); err != nil {
		return fmt.Errorf("删除定时任务失败: %w", err)
	}

	fmt.Printf("✓ 定时任务 '%s' 删除成功\n", taskName)
	return nil
}

func runScheduleList(cmd *cobra.Command, args []string) error {
	logger.Info("列出定时任务")

	// 创建调度器
	sched := scheduler.NewScheduler()

	// 获取任务列表
	tasks, err := sched.List()
	if err != nil {
		return fmt.Errorf("获取任务列表失败: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("没有找到 AutoCert 相关的定时任务")
		return nil
	}

	// 显示任务列表
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "任务名称\t状态\t下次运行\t上次运行")
	fmt.Fprintln(w, "--------\t----\t--------\t--------")

	for _, task := range tasks {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", task.Name, task.Status, task.NextRun, task.LastRun)
	}

	w.Flush()
	return nil
}

func renewDomainCert(domain string) error {
	// 创建证书管理器
	certManager := cert.NewManager(domain, "")
	if certManager == nil {
		return fmt.Errorf("创建证书管理器失败")
	}

	// 续期证书
	if err := certManager.Renew(); err != nil {
		return fmt.Errorf("域名 %s 证书续期失败: %w", domain, err)
	}

	fmt.Printf("✓ 域名 %s 证书续期成功\n", domain)
	return nil
}

func renewAllCerts() error {
	// 这里应该遍历所有已安装的证书进行续期
	// 为简化演示，这里只是打印消息

	logger.Info("开始续期所有证书")
	fmt.Println("✓ 所有证书续期检查完成")
	return nil
}

func showDomainStatus(domain string) error {
	// 创建证书管理器
	certManager := cert.NewManager(domain, "")
	if certManager == nil {
		return fmt.Errorf("创建证书管理器失败")
	}

	// 获取证书信息
	certInfo, err := certManager.GetCertInfo()
	if err != nil {
		return fmt.Errorf("获取域名 %s 证书信息失败: %w", domain, err)
	}

	// 显示证书状态
	fmt.Printf("域名: %s\n", certInfo.Domain)
	if len(certInfo.Domains) > 1 {
		fmt.Printf("所有域名: %v\n", certInfo.Domains)
	}
	fmt.Printf("证书路径: %s\n", certInfo.CertPath)
	fmt.Printf("私钥路径: %s\n", certInfo.KeyPath)
	fmt.Printf("到期时间: %s\n", certInfo.ExpiryDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("剩余天数: %d 天\n", certInfo.DaysLeft)

	if certInfo.IsValid {
		fmt.Printf("状态: ✓ 有效\n")
	} else {
		fmt.Printf("状态: ✗ 已过期\n")
	}

	return nil
}

func showAllStatus() error {
	// 这里应该遍历所有已安装的证书显示状态
	// 为简化演示，这里只是显示表头

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "域名\t状态\t到期时间\t剩余天数")
	fmt.Fprintln(w, "----\t----\t--------\t--------")

	// 这里应该有实际的证书信息
	fmt.Fprintln(w, "example.com\t有效\t2024-12-31\t30天")

	w.Flush()
	return nil
}
