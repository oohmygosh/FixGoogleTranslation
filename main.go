package main

import (
	"bufio"
	"fmt"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Ip struct {
	ip string
	ms int
}

func main() {
	println("获取最新Ip")
	ips, err := ReadTxtFormInternet("https://raw.githubusercontent.com/hcfyapp/google-translate-cn-ip/main/ips.txt")
	if err != nil {
		printRed("获取失败，读取本地文件")
		ips, err = readFile("ips.txt")
		if err != nil {
			println("按任意键退出")
			b := make([]byte, 1)
			_, _ = os.Stdin.Read(b)
			return
		}
	}
	var wg sync.WaitGroup
	wg.Add(len(ips))
	for _, i := range ips {
		go func(ip *Ip) {
			defer wg.Done()
			ms, ok := ping(ip.ip)
			if ok {
				ip.ms = ms
			}
		}(i)
	}
	wg.Wait()
	println("========================结果=========================")
	fast := ips[0]
	for _, ip := range ips {
		if ip.ms < fast.ms {
			fast = ip
		}
	}
	// translate.googleapis.com
	println("IP:", fast.ip, "延迟:", fast.ms)
	modifyHosts(fast)
	println("修改完成，按任意键退出")
	b := make([]byte, 1)
	_, _ = os.Stdin.Read(b)
}

func modifyHosts(ip *Ip) {
	hostsPath := "C:\\Windows\\System32\\drivers\\etc\\hosts"
	open, err := os.Open(hostsPath)
	if err != nil {
		log.Fatal("文件打开失败")
	}
	defer func(open *os.File) {
		_ = open.Close()
	}(open)
	reader := bufio.NewReader(open)
	host := "translate.googleapis.com"
	result := ""
	for true {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		readLine := string(line)
		if !strings.Contains(readLine, "#") && strings.Contains(readLine, host) {
			result += ip.ip + "\t" + host
		} else {
			result += readLine
		}
		result += "\n"
	}

	fw, err := os.OpenFile(hostsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666) //os.O_TRUNC清空文件重新写入，否则原文件内容可能残留
	w := bufio.NewWriter(fw)
	_, err = w.WriteString(result)
	if err != nil {
		return
	}
	if err != nil {
		panic(err)
	}
	err = w.Flush()
	if err != nil {
		printRed("修改hosts失败！请用管理员权限运行！")
		return
	}
	defer func(fw *os.File) {
		_ = fw.Close()
	}(fw)
}

func readFile(path string) ([]*Ip, error) {
	var result []*Ip
	file, err := os.Open(path)
	if err != nil {
		printRed("打开文件失败，请下载文件")
		OpenUri("https://raw.githubusercontent.com/Ponderfly/GoogleTranslateIpCheck/master/src/GoogleTranslateIpCheck/GoogleTranslateIpCheck/ip.txt")
		return result, err
	}
	reader := bufio.NewReader(file)
	for {
		ips, err := reader.ReadString('\n')
		ip := strings.TrimSpace(ips)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		result = append(result, &Ip{ip: ip, ms: 888})
		if err == io.EOF {
			break
		}
	}
	return result, nil
}

func ping(ip string) (int, bool) {
	if ip == "" {
		return 0, false
	}
	out, _ := exec.Command("ping", ip, "-n", "1").Output()
	str := ConvertGBKByte2Str(out)
	compile := regexp.MustCompile("[\\s\\S.]*平均\\s=\\s(\\d+)")
	subMatch := compile.FindStringSubmatch(str)
	if len(subMatch) < 1 {
		println("IP:", ip, "超时")
		return 0, false
	}
	ms, err := strconv.Atoi(subMatch[1])
	if err != nil {
		return 0, false
	}
	println("IP:", ip, "延迟", ms)
	return ms, true
}

func ConvertGBKByte2Str(gbkStr []byte) string {
	//如果是[]byte格式的字符串，可以使用Bytes方法
	b, _ := simplifiedchinese.GBK.NewDecoder().Bytes(gbkStr)
	return string(b)
}

func ReadTxtFormInternet(filePath string) ([]*Ip, error) {
	var result []*Ip
	resp, err := http.Get(filePath)
	if err != nil {
		return result, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("流关闭失败\n%s", err)
		}
	}(resp.Body)
	reader := bufio.NewReaderSize(resp.Body, 1024*32)
	for {
		b, errR := reader.ReadBytes('\n') //按照行读取，遇到\n结束读取
		if errR != nil {
			if errR == io.EOF {
				break
			}
			fmt.Println(errR.Error())
		}
		lineData := strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r")
		result = append(result, &Ip{ip: lineData, ms: 999})
	}

	return result, nil
}

func OpenUri(uri string) {
	cmd := exec.Command(`cmd`, `/c`, `start`, uri)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err := cmd.Start()
	if err != nil {
		return
	}
}

func printRed(str string) {
	fmt.Printf("%c[1;40;31m%s%c[0m\n", 0x1B, str, 0x1B)
}
