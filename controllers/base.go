package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/streadway/amqp"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
)

type BaseController struct {
	beego.Controller
}

type ReturnMsg struct {
	Code int
	Msg  string
	Data interface{}
}

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	//队列名称
	QueueName string
	//交换机名称
	Exchange string
	//bind Key 名称
	Key string
	//连接信息
	Mqurl string
}

type LoggerConfig struct {
	FileName            string `json:"filename"` //将日志保存到的文件名及路径
	Level               int    `json:"level"`    // 日志保存的时候的级别，默认是 Trace 级别
	Maxlines            int    `json:"maxlines"` // 每个文件保存的最大行数，若文件超过maxlines，则将日志保存到下个文件中，为0表示不设置。默认值 1000000
	Maxsize             int    `json:"maxsize"`  // 每个文件保存的最大尺寸，若文件超过maxsize，则将日志保存到下个文件中，为0表示不设置。默认值是 1 << 28, //256 MB
	Daily               bool   `json:"daily"`    // 设置日志是否每天分割一次，默认是 true
	Maxdays             int    `json:"maxdays"`  // 设置保存最近几天的日志文件，超过天数的日志文件被删除，为0表示不设置，默认保存 7 天
	Rotate              bool   `json:"rotate"`   // 是否开启 logrotate，默认是 true
	Perm                string `json:"perm"`     // 日志文件权限
	RotatePerm          string `json:"rotateperm"`
	EnableFuncCallDepth bool   `json:"-"` // 输出文件名和行号
	LogFuncCallDepth    int    `json:"-"` // 函数调用层级
}

/*
@desc get请求
@param url string 请求地址
@param timeout int 超时时间
@return string
*/
func (this *BaseController)CurlGet(url string)(ret string,err error){
	ret = ""
	client := http.Client{}
	responseHtml,err :=client.Get(url)
	if err != nil {
		return
	}
	defer responseHtml.Body.Close()
	jsonStr,_ := ioutil.ReadAll(responseHtml.Body)
	ret = string(jsonStr)
	return
}

/*
post请求
@param url string 请求地址
@param data maxStruct 需要传递的参数
@return string 接口返回的数据
*/
func (this *BaseController)Curlpost(url string, reqmsg map[string]interface{}) (ret map[string]interface{},err error) {
	dataType, _ := json.Marshal(reqmsg)
	dataString := string(dataType)
	fmt.Println(dataString)
	client := &http.Client{}
	res, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(dataString)))
	if err != nil {
		return
	}
	res.Header.Add("Content-Type", "application/json;charset=utf-8")
	response, err := client.Do(res)
	defer response.Body.Close()
	jsonStr, _ := ioutil.ReadAll(response.Body)
	if err = json.Unmarshal(jsonStr,&ret);err != nil{
		return
	}
	return
}

/*
获取华为token
@return string token
*/
func(this *BaseController)GetHuaWeiToken()(token string){
	var tokenurl = beego.AppConfig.String("insidesite")+`sotu/gethuaweitoken`
	tokenrst,_:=this.CurlGet(tokenurl)
	token=gjson.Get(tokenrst,"data.token").String()
	return token
}

/*
httpform接口返回
 */
func (this *BaseController)HttpPostForm(url string,postdata map[string][]string)(ret map[string]interface{},err error) {
	resp, err := http.PostForm(url,
		postdata)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	jsonStr,_ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(jsonStr,&ret);err != nil{
		return
	}
	return
}

/*
请求成功返回数据
*/
func (this *BaseController) SuccessJson(data interface{}) {
	res := ReturnMsg{
		200, "success", data,
	}
	this.Data["json"] = res
	this.ServeJSON() //对json进行序列化输出
	this.StopRun()
}

/*
请求失败返回数据
*/
func (this *BaseController) ErrorJson(code int, msg string, data interface{}) {
	res := ReturnMsg{
		code, msg, data,
	}
	this.Data["json"] = res
	this.ServeJSON() //对json进行序列化输出
	this.StopRun()
}

func (this *BaseController)NewRabbitMQ(MQURL string,queueName string, exchange string, key string) *RabbitMQ {
	return &RabbitMQ{QueueName: queueName, Exchange: exchange, Key: key, Mqurl: MQURL}
}

func (this *BaseController)SiteLogs(filename string,msg string,logtype int){
	var logCfg = LoggerConfig{
		FileName:            beego.AppConfig.String("LogsPath")+filename,
		Level:               logs.LevelDebug,
		Daily:               true,
		EnableFuncCallDepth: false,
		LogFuncCallDepth:    3,
		RotatePerm:          "777",
		Perm:                "777",
	}

	// 设置beego log库的配置
	b, _ := json.Marshal(&logCfg)
	logs.SetLogger(logs.AdapterFile, string(b))
	if logtype==1 {
		logs.Info(msg)
	}else{
		logs.Error(msg)
	}
	logs.Async()
}

/*
发送钉钉消息
*/
func (this *BaseController)Dingding(msg string) {
	fmt.Println("err",msg)
	return
	url := "https://oapi.dingtalk.com/robot/send?access_token=a7ac389ba1b08e15245b255f6c7efdcfcc27e7f3ae1dc71a44a9cea1e311a389"
	var data map[string]interface{}
	var text map[string]string
	var at map[string]string
	data = make(map[string]interface{})
	data["msgtype"] = "text"
	text = make(map[string]string)
	text["content"] = "【转版本】" + msg
	data["text"] = text
	at = make(map[string]string)
	at["isAtAll"] = "true"
	data["at"] = at
	_,_=this.Curlpost(url, data)
}