package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/fatedier/frp/g"
	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server"
	"github.com/fatedier/frp/utils/log"
	"github.com/fatedier/frp/utils/util"
	"github.com/fatedier/frp/utils/version"
	"github.com/fatedier/frp/utils/vhost" // 添加 vhost 包的导入
)

const (
	CfgFileTypeIni = iota
	CfgFileTypeCmd
)

var (
	cfgFile     string
	showVersion bool

	bindAddr                   string
	bindPort                   int
	bindUdpPort                int
	kcpBindPort                int
	proxyBindAddr              string
	vhostHttpPort              int
	vhostHttpsPort             int
	vhostHttpTimeout           int64
	dashboardAddr              string
	dashboardPort              int
	dashboardUser              string
	dashboardPwd               string
	logFile                    string
	logLevel                   string
	logMaxDays                 int64
	token                      string
	subDomainHost              string
	tcpMux                     bool
	allowPorts                 string
	maxPoolCount               int64
	maxPortsPerClient          int64
	maxServiceUnavailableCount uint64
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file of frps")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of frps")

	rootCmd.PersistentFlags().StringVarP(&bindAddr, "bind_addr", "", "0.0.0.0", "bind address")
	rootCmd.PersistentFlags().IntVarP(&bindPort, "bind_port", "p", 7000, "bind port")
	rootCmd.PersistentFlags().IntVarP(&bindUdpPort, "bind_udp_port", "", 0, "bind udp port")
	rootCmd.PersistentFlags().IntVarP(&kcpBindPort, "kcp_bind_port", "", 0, "kcp bind udp port")
	rootCmd.PersistentFlags().StringVarP(&proxyBindAddr, "proxy_bind_addr", "", "0.0.0.0", "proxy bind address")
	rootCmd.PersistentFlags().IntVarP(&vhostHttpPort, "vhost_http_port", "", 0, "vhost http port")
	rootCmd.PersistentFlags().IntVarP(&vhostHttpsPort, "vhost_https_port", "", 0, "vhost https port")
	rootCmd.PersistentFlags().Int64VarP(&vhostHttpTimeout, "vhost_http_timeout", "", 60, "vhost http response header timeout")
	rootCmd.PersistentFlags().StringVarP(&dashboardAddr, "dashboard_addr", "", "0.0.0.0", "dashboard address")
	rootCmd.PersistentFlags().IntVarP(&dashboardPort, "dashboard_port", "", 0, "dashboard port")
	rootCmd.PersistentFlags().StringVarP(&dashboardUser, "dashboard_user", "", "admin", "dashboard user")
	rootCmd.PersistentFlags().StringVarP(&dashboardPwd, "dashboard_pwd", "", "admin", "dashboard password")
	rootCmd.PersistentFlags().StringVarP(&logFile, "log_file", "", "console", "log file")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log_level", "", "info", "log level")
	rootCmd.PersistentFlags().Int64VarP(&logMaxDays, "log_max_days", "", 3, "log max days")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "auth token")
	rootCmd.PersistentFlags().StringVarP(&subDomainHost, "subdomain_host", "", "", "subdomain host")
	rootCmd.PersistentFlags().StringVarP(&allowPorts, "allow_ports", "", "", "allow ports")
	rootCmd.PersistentFlags().Int64VarP(&maxPortsPerClient, "max_ports_per_client", "", 0, "max ports per client")
	rootCmd.PersistentFlags().Uint64VarP(&maxServiceUnavailableCount, "max_service_unavailable_count", "m", 10, "max service unavailable count")
}

var rootCmd = &cobra.Command{
	Use:   "frps",
	Short: "frps is the server of frp (https://github.com/fatedier/frp)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			fmt.Println(version.Full())
			return nil
		}

		var err error
		if cfgFile != "" {
			var content string
			content, err = config.GetRenderedConfFromFile(cfgFile)
			if err != nil {
				return err
			}
			g.GlbServerCfg.CfgFile = cfgFile
			err = parseServerCommonCfg(CfgFileTypeIni, content)
		} else {
			err = parseServerCommonCfg(CfgFileTypeCmd, "")
		}
		if err != nil {
			return err
		}

		err = runServer()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func parseServerCommonCfg(fileType int, content string) (err error) {
	if fileType == CfgFileTypeIni {
		err = parseServerCommonCfgFromIni(content)
	} else if fileType == CfgFileTypeCmd {
		err = parseServerCommonCfgFromCmd()
	}
	if err != nil {
		return
	}

	err = g.GlbServerCfg.ServerCommonConf.Check()
	if err != nil {
		return
	}

	config.InitServerCfg(&g.GlbServerCfg.ServerCommonConf)

	// 传递 maxServiceUnavailableCount 到 resource.go
	vhost.SetMaxServiceUnavailableCount(g.GlbServerCfg.ServerCommonConf.MaxServiceUnavailableCount)

	return
}

func parseServerCommonCfgFromIni(content string) (err error) {
	cfg, err := config.UnmarshalServerConfFromIni(&g.GlbServerCfg.ServerCommonConf, content)
	if err != nil {
		return err
	}
	g.GlbServerCfg.ServerCommonConf = *cfg
	return
}

func parseServerCommonCfgFromCmd() (err error) {
	g.GlbServerCfg.BindAddr = bindAddr
	g.GlbServerCfg.BindPort = bindPort
	g.GlbServerCfg.BindUdpPort = bindUdpPort
	g.GlbServerCfg.KcpBindPort = kcpBindPort
	g.GlbServerCfg.ProxyBindAddr = proxyBindAddr
	g.GlbServerCfg.VhostHttpPort = vhostHttpPort
	g.GlbServerCfg.VhostHttpsPort = vhostHttpsPort
	g.GlbServerCfg.VhostHttpTimeout = vhostHttpTimeout
	g.GlbServerCfg.DashboardAddr = dashboardAddr
	g.GlbServerCfg.DashboardPort = dashboardPort
	g.GlbServerCfg.DashboardUser = dashboardUser
	g.GlbServerCfg.DashboardPwd = dashboardPwd
	g.GlbServerCfg.LogFile = logFile
	g.GlbServerCfg.LogLevel = logLevel
	g.GlbServerCfg.LogMaxDays = logMaxDays
	g.GlbServerCfg.Token = token
	g.GlbServerCfg.SubDomainHost = subDomainHost
	g.GlbServerCfg.MaxServiceUnavailableCount = maxServiceUnavailableCount
	if len(allowPorts) > 0 {
		// e.g. 1000-2000,2001,2002,3000-4000
		ports, errRet := util.ParseRangeNumbers(allowPorts)
		if errRet != nil {
			err = fmt.Errorf("Parse conf error: allow_ports: %v", errRet)
			return
		}

		for _, port := range ports {
			g.GlbServerCfg.AllowPorts[int(port)] = struct{}{}
		}
	}
	g.GlbServerCfg.MaxPortsPerClient = maxPortsPerClient

	if logFile == "console" {
		g.GlbServerCfg.LogWay = "console"
	} else {
		g.GlbServerCfg.LogWay = "file"
	}
	return
}

func runServer() (err error) {
	log.InitLog(g.GlbServerCfg.LogWay, g.GlbServerCfg.LogFile, g.GlbServerCfg.LogLevel,
		g.GlbServerCfg.LogMaxDays)
	svr, err := server.NewService()
	if err != nil {
		return err
	}
	log.Info("Start frps success")
	server.ServerService = svr
	svr.Run()
	return
}
