package main

import (
	"encoding/json"
	"fmt"
	"github.com/kardianos/service"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

//todo
//1 只要服务处于停止状态，守护进程就会反复重启
//2 如何让守护进程被安装时，读到当前位置, 当前只能用绝对路径

var logger service.Logger

type program struct{}

var cfg *Config

func main() {
	cfg = GetConfig()

	svcConfig := &service.Config{
		Name:        "Go ServiceWatcher",
		DisplayName: "Go ServiceWatcher",
		Description: "This is an example Go service about server watch.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}
func (p *program) run() {
	fmt.Println("server watch")
	NewTicker(cfg.Interval)
	//select主要用来监控多个channel，channel的数据读取，写入，关闭等事件，采用的是轮训算法
	select {}
}
func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

//Ticker是一个定时触发的计时器，
//它会以一个间隔(interval)往channel发送一个事件(当前时间)，
//而channel的接收者可以以固定的时间间隔从channel中读取事件
func NewTicker(interval time.Duration) {
	ticker := time.NewTicker(time.Second * interval)

	i := 0
	go func() {
		for {
			<-ticker.C
			i++
			t := time.Now().Format("2006-01-02 15:04:05")
			fmt.Println("i = ", i, "time = ", t)
			//服务名大小写敏感
			as := []string{
				cfg.ServerName,
				//"Appinfo",
				//"appinfo",
				//"Ssit.SmartBookShelf",
				//"xbgm",
				//"xxxx",
			}
			rt := CheckServiceWorking(as)
			StartServer(rt)
		}
	}()
	return
}

func CheckServiceWorking(checks []string) map[string]bool {
	m, _ := mgr.Connect()
	s, _ := m.ListServices()
	defer m.Disconnect()
	rt := map[string]bool{}
	for _, v := range checks {
		rt[v] = false
	}

	for _, v := range s {
		for _, c := range checks {
			if c == v {
				srv, _ := m.OpenService(c)
				defer srv.Close()
				srvStatus, _ := srv.Query()
				if srvStatus.State == windows.SERVICE_RUNNING && CheckWebApiWorking(cfg.URL) {
					rt[v] = true
					continue
				}
			}
		}
	}

	return rt
}

func StartServer(servers map[string]bool) {
	for k, v := range servers {
		fmt.Println("service ", k, " status is ", v)
		if v == false {
			fmt.Println("restart service ", k)
			cmd1 := exec.Command("net", "stop", k)
			fmt.Println(cmd1)
			if out1, err := cmd1.CombinedOutput(); err != nil {
				fmt.Println(string(out1), err)
			}
			time.Sleep(time.Second * 2)
			cmd2 := exec.Command("net", "start", k)
			fmt.Println(cmd2)
			if out2, err := cmd2.CombinedOutput(); err != nil {
				fmt.Println(string(out2), err)
			}
			//m, _ := mgr.Connect()
			//defer m.Disconnect()
			//srv, _ := m.OpenService(k)
			//defer srv.Close()
			//srv.Close()
			//srv.Start()
		}
	}
}

type Config struct {
	ServerName string        `json:"serverName"`
	Interval   time.Duration `json:"interval"`
	URL        string        `json:"url"`
}

// 创建一个错误处理函数，避免过多的 if err != nil{} 出现
func dropErr(e error) {
	if e != nil {
		fmt.Println(e)
		panic(e)
	}
}

const CONFIG_PATH = "config.json"

//const CONFIG_PATH  = "D:\\GoPath\\src\\town\\TestTools\\ServerWatcher\\config.json"
func GetConfig() (cfg *Config) {

	fpt, err := os.Getwd()
	//fpt,err := filepath.Abs(filepath.Dir(CONFIG_PATH))
	dropErr(err)
	//ioutil读取整个文件
	fileData, err := ioutil.ReadFile(fpt + "\\" + CONFIG_PATH)
	dropErr(err)
	cfgStr := string(fileData)
	//fmt.Println(cfgStr)

	// bufio 读取
	//f,err := os.Open(CONFIG_PATH)
	//dropErr(err)
	//bio:=bufio.NewReader(f)
	// ReadLine() 方法一次尝试读取一行，如果过默认缓存值就会报错。默认遇见'\n'换行符会返回值。isPrefix 在查找到行尾标记后返回 false
	//bfRead,isPrefix,err:=bio.ReadLine()
	//dropErr(err)
	//fmt.Println("This mess is  [ %q ] [%v]\n", bfRead, isPrefix)
	//str := `{"Configs":[{"ServerName":"Shanghai_VPN","ServerIp":"127.0.0.1"},
	//		{"ServerName":"Beijing_VPN","ServerIp":"127.0.0.2"}]}`

	//var c Config
	c := &Config{}
	json.Unmarshal([]byte(cfgStr), &c)
	fmt.Println(c.ServerName, c.Interval, c.URL)
	return c
}

func CheckWebApiWorking(url string) (rt bool) {
	rsp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer rsp.Body.Close()
	fmt.Println(rsp.Status)
	return rsp.StatusCode == http.StatusOK
}
