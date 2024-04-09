package main

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int
	// 在线用户的map表
	OnlineMap map[string]*User // key放当前用户名，value存放user对象
	// 由于map是全局的，需要加一个读写锁
	mapLock sync.RWMutex // 同步的全部机制都在sync包中
	// 消息广播的channel
	Message chan string
}

// 创建一个server接口，即返回server对象。NewServer作为一个类对外开放的方法
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息channel的goroutine，一旦有消息就发送给全部在线的User
func (ser *Server) ListenMessager() {
	for {
		msg := <-ser.Message // 不断尝试从Message channel中读数据
		// 将msg发送给全部在线的User
		ser.mapLock.Lock()
		for _, cli := range ser.OnlineMap { // 从map值中拿到user对象
			cli.C <- msg
		}
		ser.mapLock.Unlock()
	}
}

// 广播消息的方法，参数包括：由哪个用户发起、消息内容是什么
func (ser *Server) BroadCast(user *User, msgType string, msg string) {
	sendMsg := "[" + msgType + "]" + user.Name + ":" + msg // 消息内容 做一个字符串拼接
	ser.Message <- sendMsg                                 // 将消息发给Message的channel中
}

func (ser *Server) Handler(conn net.Conn) {
	//fmt.Println("连接建立成功")
	user := NewUser(conn, ser)
	// 广播用户上线
	user.Online()
	// 给新人发送新人指引
	user.SendHelp()

	// 监听用户是否活跃的channel
	isAlive := make(chan bool)

	// 启动goroutine来接受客户端发送的消息
	go func() {
		// 创建一个4kb的缓冲区，一般足够存储一次性读取的文本消息
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf) // 从连接中读取数据到 buf 中，并返回读取的字节数 n 和可能发生的错误 err
			if n == 0 {
				// 说明客户端断开了连接，广播用户已下线
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("Conn Read err:", err)
				return
			}

			// 提取用户的消息（去除结尾的'\n'）
			msg := string(buf[:n-1])
			// 将得到的消息进行广播
			user.DoMessage(msg)

			// 用户的任意消息，都会使用户处于活跃
			isAlive <- true
		}
	}()

	// 超时踢出
	// 科普解释下select原理：每次执行到 select 语句时，所有的 case 条件都会被重新评估，包括任何调用 time.After 的 case。time.After 函数返回一个 chan Time 类型的单向通道，这个通道会在指定时间后接收到一个时间值。
	// 重要的是，每次 select 语句包含的 time.After 被执行时，都会创建一个新的定时器实例。这就意味着，如果 select 语句在定时器触发前被重新执行（比如因为其他 case 被选中执行），旧的定时器将被丢弃（不再引用），并且一个新的定时器开始计时。
	// 所以，当 select 通过 <-isAlive 激活时，这个动作实际上导致了 select 语句的重新执行，因为这通常发生在一个循环中。一旦 select 重新执行，case <-time.After(time.Second * 10): 这一行也会重新执行，从而启动了一个全新的定时器。这就实现了每次用户活跃时重置超时计时的效果。
	for {
		// 监听不同channel，保持当前goroutine存活
		select {
		// 定时器活跃检测
		case <-isAlive:
			// 当前用户活跃，无需处理，激活select更新下面case中的定时器
		case <-time.After(time.Minute * 5): // 指定5分钟
			// 已超时，将当前user客户端强制关闭
			user.SendMsg("系统", "您潜水太久已超时，请重新连接进入")
			// 销毁channel资源
			close(user.C)
			// 关闭绑定Handler的连接
			conn.Close()
			// 退出当前Handler goroutine
			runtime.Goexit() // 或 return
		}
	}
}

// 启动服务器接口
func (ser *Server) Start() { // ser是创建了一个类对象，便于使用类的属性
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ser.Ip, ser.Port)) // Sprintf用于拼接字符串，拼接为"127.0.0.1:8888"
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	//close listen socket
	defer listener.Close() // 为了防止遗忘关闭，使用defer关闭

	//启动监听Message的goroutine
	go ser.ListenMessager()

	for {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener Accept err:", err)
			continue
		}
		//do handler 业务回调
		go ser.Handler(conn)
	}
}
