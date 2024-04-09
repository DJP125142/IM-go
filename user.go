package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string // 跟用户绑定的channel
	conn   net.Conn    // 当前用户和客户端通信的连接句柄
	server *Server     // 关联上当前服务指针
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String() // 获取用户地址（IP和端口）
	// 实例化一个用户
	user := &User{ // 获取User对象
		Name:   userAddr, // 获取当前客户端地址
		Addr:   userAddr,
		C:      make(chan string), // 每个用户创建一个新的channel
		conn:   conn,
		server: server,
	}
	// 启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (u *User) Online() {
	u.server.mapLock.Lock()        // 加锁
	u.server.OnlineMap[u.Name] = u // key是用户名，value是用户对象
	u.server.mapLock.Unlock()      // 释放锁
	// 广播当前用户上线消息
	u.server.BroadCast(u, "系统", "已上线")
}

// 新人指引
func (u *User) SendHelp() {
	// 给上线用户发送操作指引
	u.SendMsg("系统", "发送“rename|{your name}”可以修改昵称,例如“rename|张三”\n[系统]使用“who”查看当前在线人员\n[系统]使用“to|张三|你好呀”可发起私聊\n")
}

// 用户下线业务
func (u *User) Offline() {
	u.server.mapLock.Lock() // 加锁
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock() // 释放锁
	// 广播当前用户上线消息
	u.server.BroadCast(u, "系统", "已下线")
}

// 用户处理消息的业务
func (u *User) DoMessage(msg string) {
	switch {
	case msg == "who":
		// ---------特殊指令who------------
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := user.Name + ":" + "在线ing\n"
			u.SendMsg("系统", onlineMsg)
		}
		u.server.mapLock.Unlock()
	case strings.HasPrefix(msg, "rename|") && len(msg) > 7:
		// ---------特殊指令rename|------------
		// 重命名的逻辑
		newName := strings.Split(msg, "|")[1]
		_, ok := u.server.OnlineMap[newName]
		if ok {
			// 用户名被占用
			u.SendMsg("系统", "当前用户名已被占用\n")
		} else {
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name) // 删除旧用户名
			u.server.OnlineMap[newName] = u    // 添加新用户名
			u.server.mapLock.Unlock()

			u.Name = newName // 更新用户对象中的用户名
			u.SendMsg("系统", "您已修改用户名为："+u.Name+"\n")
		}
	case len(msg) > 4 && msg[:3] == "to|":
		// 消息格式：to|张三|你好啊
		//1 获取对方的用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.SendMsg("系统", "消息格式不正确，请使用\"to|张三|你好啊\"格式\n")
			return
		}
		//2 根据用户名得到对方user对象
		remoteUser, ok := u.server.OnlineMap[remoteName]
		if !ok { // 该用户名不存在/不在线
			u.SendMsg("系统", "该用户名不存在/不在线\n")
			return
		}
		//3 获取消息内容，通过对方的user对象将消息内容发送过去
		content := strings.Split(msg, "|")[2]
		if content == "" {
			u.SendMsg("系统", "无消息内容，请重发\n")
		}
		remoteUser.SendMsg("私聊", u.Name+"对您说："+content)
	case msg == "":
		u.SendMsg("系统", "无消息内容，请重发\n")
	default:
		// 默认广播消息
		u.server.BroadCast(u, "大厅", msg)
	}
}

// 私聊给当前user的客户端发送消息
func (u *User) SendMsg(msgType string, msg string) {
	u.conn.Write([]byte("[" + msgType + "]" + msg))
}

// 监听当前user channel的方法，一旦有消息，就发送给对端客户端
func (u *User) ListenMessage() {
	// 在无限循环中从该用户的channel中读取消息
	for {
		msg := <-u.C                     // 从channel中读取数据到msg
		u.conn.Write([]byte(msg + "\n")) // 使用byte数组来写msg，通过conn发送给客户端
	}
}
