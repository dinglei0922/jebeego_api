package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/streadway/amqp"
	"time"
)

var queueName = "footprint"
var logName = "footprint.log"

type FootPrintMQController struct {
	BaseController
}



func (c *FootPrintMQController)NewRabbit()*RabbitMQ{
	dialstr := fmt.Sprintf("amqp://%v:%v@%v",beego.AppConfig.String("footprintmq::footmq_name"),beego.AppConfig.String("footprintmq::footmq_pwd"),beego.AppConfig.String("footprintmq::footmq_host"))
	return c.NewRabbitMQ(dialstr,queueName, "", "")
}

func init(){
	go func() {
		foot:=FootPrintMQController{}
		foot.MQConsume()
	}()
}

func (c *FootPrintMQController)MQPublish() {
	rabbit:=c.NewRabbit()
	var err error
	rabbit.conn, err = amqp.Dial(rabbit.Mqurl)
	if err != nil {
		c.ErrorInfo(fmt.Sprintf("生成者-RabbitMQ连接失败:%v",err))
	}
	// 创建一个channel
	rabbit.channel, err = rabbit.conn.Channel()
	if err != nil {
		c.ErrorInfo(fmt.Sprintf("生成channel打开失败:%v",err))
	}
	defer rabbit.channel.Close()
	mqcontent := c.GetString("mqcontent")
	// 声明一个队列
	q, err := rabbit.channel.QueueDeclare(
		queueName, // 队列名称
		true,      // 是否持久化
		false,     // 是否自动删除
		false,     // 是否独立
		false,
		nil,
	)

	if err != nil {
		c.ErrorInfo(fmt.Sprintf("声明队列失败:%v",err))
	}
	// 发送消息到队列中
	err = rabbit.channel.Publish(
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
		c.ErrorInfo(fmt.Sprintf("生成者-发送消息到队列失败:%v",err))
	}
}

/*
足迹队列消费
*/
func (c *FootPrintMQController)MQConsume() {
	defer func() {
		if err := recover(); err != nil {
			time.Sleep(3 * time.Second)
			c.MQConsume()
			c.ErrorInfo(fmt.Sprintf("足迹消费错误:%v",err))
		}
	}()
	// 接收参数
	rabbit:=c.NewRabbit()
	var err error
	rabbit.conn, err = amqp.Dial(rabbit.Mqurl)
	if err != nil {
		time.Sleep(3 * time.Second)
		c.MQConsume()
		c.ErrorInfo(fmt.Sprintf("足迹消费amqp.Dial生成错误:%v",err))
	}
	rabbit.channel,err = rabbit.conn.Channel()
	if err != nil {
		time.Sleep(3 * time.Second)
		c.MQConsume()
		c.ErrorInfo(fmt.Sprintf("足迹消费Channel生成错误:%v",err))
	}
	q, err := rabbit.channel.QueueDeclare(
		rabbit.QueueName,
		//是否持久化
		true,
		//是否自动删除
		false,
		//是否具有排他性
		false,
		//是否阻塞处理
		false,
		//额外的属性
		nil,
	)
	if err != nil {
		time.Sleep(3 * time.Second)
		c.MQConsume()
		c.ErrorInfo(fmt.Sprintf("足迹消费QueueDeclare生成错误:%v",err))
	}

	//消费者流控
	rabbit.channel.Qos(
		1, //当前消费者一次能接受的最大消息数量
		0, //服务器传递的最大容量（以八位字节为单位）
		false, //如果设置为true 对channel可用
	)

	//接收消息
	msgs, err := rabbit.channel.Consume(
		q.Name, // queue
		//用来区分多个消费者
		"", // consumer
		//是否自动应答
		//这里要改掉，我们用手动应答
		false, // auto-ack
		//是否独有
		false, // exclusive
		//设置为true，表示 不能将同一个Conenction中生产者发送的消息传递给这个Connection中 的消费者
		false, // no-local
		//列是否阻塞
		false, // no-wait
		nil,   // args
	)

	forever := make(chan bool)
	go func() {
		closeChan := make(chan *amqp.Error, 1)
		notifyClose := rabbit.channel.NotifyClose(closeChan)
		//一旦消费者的channel有错误，产生一个amqp.Error，channel监听并捕捉到这个错误
		closeFlag := false
		for {
			select {
			case e := <-notifyClose:
				close(closeChan)
				time.Sleep(3 * time.Second)
				c.MQConsume()
				c.SiteLogs(logName,fmt.Sprintf("消费禅道出错：%v",e.Error()),2)
				closeFlag = true
			case msg := <-msgs:
				c.SiteLogs(logName,fmt.Sprintf("消费了 msg:%v",string(msg.Body)),1)
				msg.Ack(false)
			}
			if closeFlag {
				break
			}
		}
	}()
	<-forever
}

/*
请求task接口完成足迹添加
*/
func (c *FootPrintMQController)Sendtaskfootprint(postdata []byte){
	fmt.Println(string(postdata))


	return
	//url := beego.AppConfig.String("tasksite")+"home/footprint/addfootprint.php"
	//ret ,_:= c.Curlpost(url, postdata)
	//if status,ok := ret["status"].(int);!ok || status!=200{
	//	logs.Info("足迹添加失败")
	//}
}

func (c *FootPrintMQController)ErrorInfo(msg string){
	c.SiteLogs("footprint.log",msg,2)
	c.Dingding(msg)
	c.Data["json"] = map[string]interface{}{"status":500,"msg":msg}
	c.ServeJSON()
}


