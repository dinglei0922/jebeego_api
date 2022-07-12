package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/streadway/amqp"
	"time"
)

var queueName = "justeasy_footprint"

var amqp_dial struct{
	conn *amqp.Connection
	err error
}

type FootPrintMQController struct {
	BaseController
}

func (c *FootPrintMQController)init(){
	dialstr := fmt.Sprintf("amqp://%v:%v@%v",beego.AppConfig.String("footprintmq::footmq_name"),beego.AppConfig.String("footprintmq::footmq_pwd"),beego.AppConfig.String("footprintmq::footmq_host"))
	conn,err:=amqp.Dial(dialstr)
	amqp_dial.conn = conn
	amqp_dial.err = err
}

func (c *FootPrintMQController)MQPublish() {
	conn:=amqp_dial.conn
	if amqp_dial.err != nil {
		c.Dingding("生成者-RabbitMQ连接失败")
		return
	}
	defer conn.Close()
	// 创建一个channel
	ch, err := conn.Channel()
	if err != nil {
		c.Dingding("生成者-channel打开失败")
		return
	}
	defer ch.Close()
	mqcontent := c.GetString("mqcontent")
	// 声明一个队列
	q, err := ch.QueueDeclare(
		queueName, // 队列名称
		true,      // 是否持久化
		false,     // 是否自动删除
		false,     // 是否独立
		false,
		nil,
	)

	if err != nil {
		//c.Dingding("生成者-队列连接失败")
	}
	// 发送消息到队列中
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(mqcontent),
			DeliveryMode: 2,
		})
	if err != nil {
		//c.Dingding("生成者-发送消息到队列失败")
	}
}

/*
足迹队列消费
*/
func (c *FootPrintMQController)MQConsume() {
	defer func() {
		if err := recover(); err != nil {
			time.Sleep(3 * time.Second)
			fmt.Println("休息3秒")
			//StartAMQPConsume()
		}
	}()

	// 接收参数
	consumerName := "consumer-" + time.Now().Format("2006-01-02_15-04-05")
	conn:=amqp_dial.conn
	if amqp_dial.err != nil {
		c.Dingding("生成者-RabbitMQ连接失败")
		return
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		fmt.Println(err)
	}
	defer ch.Close()
	closeChan := make(chan *amqp.Error, 1)
	notifyClose := ch.NotifyClose(closeChan)
	//一旦消费者的channel有错误，产生一个amqp.Error，channel监听并捕捉到这个错误
	closeFlag := false
	msgs, err := ch.Consume(queueName, consumerName, false, false, false, false, nil)
	go func() {
		for {
			select {
			case e := <-notifyClose:
				fmt.Println(queueName+"chan通道错误,e:%s", e.Error())
				close(closeChan)
				time.Sleep(3 * time.Second)
				closeFlag = true
			case msg := <-msgs:
				jsondata := msg.Body
				ret :=make(map[string]interface{})
				_ = json.Unmarshal(jsondata,&ret)
				c.Sendtaskfootprint(ret)
				msg.Ack(false)
			}
			if closeFlag {
				break
			}
		}
	}()
}

/*
请求task接口完成足迹添加
*/
func (c *FootPrintMQController)Sendtaskfootprint(postdata map[string]interface{}){
	url := beego.AppConfig.String("tasksite")+"home/footprint/addfootprint.php"
	ret ,_:= c.Curlpost(url, postdata)
	if status,ok := ret["status"].(int);!ok || status!=200{
		logs.Info("足迹添加失败")
	}
}


