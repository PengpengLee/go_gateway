package main

// 观察者模式
// 又叫：
//	模型（Model）-视图（View）模式、
//	源-收听者(Listener)模式
//	从属者模式
//	发布-订阅模式
//
// 一对多的依赖关系，当一个对象的状态发生改变时，所有依赖于它的对象都得到通知并被自动更新
// 一个目标对象，管理所有依赖它的观察者对象，并在它本身的状态改变（事件）时通知所有依赖对象
// 主体是通知的发布者，它发出通知时并不需要知道谁是它的观察者，可以有任意数目的观察者订阅并接收通知
//
// 观察者模式中有四个角色：
//	1、抽象主体（Subject）：
//		它把所有观察者对象的引用保存到一个集里，每个主体都可以有任何数量的观察者。
//		抽象主体提供一个接口，可以增加和删除观察者对象。
//	2、具体主体（Concrete Subject）：
//		将有关状态存入具体观察者对象；在具体主体内部状态改变时，给所有登记过的观察者发出通知。
//	3、抽象观察者（Observer）：
//		为所有的具体观察者定义一个接口，在得到主体通知时更新自己。
//	4、具体观察者（Concrete Observer）：
//		实现抽象观察者角色所要求的更新接口，以便使本身的状态与主体状态协调

// Observer 观察者接口
// 多个观察者对应一个被观察者，相当于多个监听者对应一个被监听对象
// 观察者通过 Update 方法更新自己的可用服务列表
type Observer interface {
	// Update 观察者（监听者）更新自己的可用服务器列表
	Update()
}

// ConcreteSubject 观察者具体主体
type ConcreteSubject struct {
	observers []Observer // 观察者列表
	conf      []string   // 配置信息，即数据
	name      string     // 主体名称
}

// NewConcreteSubject 根据指定名称返回一个具体主体实例
func NewConcreteSubject(name string) (*ConcreteSubject, error) {
	mConf := &ConcreteSubject{name: name}
	return mConf, nil
}

// Attach 将新的观察者添加到观察者列表
func (s *ConcreteSubject) Attach(o Observer) {
	s.observers = append(s.observers, o)
}

// NotifyAllObservers 通知所有观察者
func (s *ConcreteSubject) NotifyAllObservers() {
	for _, obs := range s.observers {
		obs.Update()
	}
}

// GetConf 获取具体主体的配置信息
func (s *ConcreteSubject) GetConf() []string {
	return s.conf
}

// UpdateConf 更新配置信息时，通知所有观察者（监听者）也更新
// 更新配置信息意味着发生了事件，所以要通知观察者
func (s *ConcreteSubject) UpdateConf(conf []string) {
	s.conf = conf
	for _, obs := range s.observers {
		obs.Update()
	}
}
