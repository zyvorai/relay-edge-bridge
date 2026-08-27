package main

import (
    "context"
    "flag"
    "log"
    "math"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gopcua/opcua/id"
    "github.com/gopcua/opcua/server"
    "github.com/gopcua/opcua/ua"
)

type logger int
func (l logger) Debug(string,...any){};func (l logger) Info(s string,a ...any){log.Printf(s,a...)};func (l logger) Warn(s string,a ...any){log.Printf(s,a...)};func (l logger) Error(s string,a ...any){log.Printf(s,a...)}

func main(){host:=flag.String("host","0.0.0.0","listen host");port:=flag.Int("port",4840,"listen port");flag.Parse();opts:=[]server.Option{server.EnableSecurity("None",ua.MessageSecurityModeNone),server.EnableAuthMode(ua.UserTokenTypeAnonymous),server.EndPoint(*host,*port),server.EndPoint("localhost",*port),server.SetLogger(logger(0))};s:=server.New(opts...);if err:=s.Start(context.Background());err!=nil{log.Fatal(err)};defer s.Close();root,_:=s.Namespace(0);ns:=server.NewNodeNameSpace(s,"RelayEdgeSimulator");root.Objects().AddRef(ns.Objects(),id.HasComponent,true);temp:=ns.AddNewVariableStringNode("Temperature",float64(72));press:=ns.AddNewVariableStringNode("Pressure",float64(13.5));rpm:=ns.AddNewVariableStringNode("RPM",float64(1800));ns.Objects().AddRef(temp,id.HasComponent,true);ns.Objects().AddRef(press,id.HasComponent,true);ns.Objects().AddRef(rpm,id.HasComponent,true);ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer cancel();go func(){t:=time.NewTicker(time.Second);defer t.Stop();phase:=0.0;for{select{case<-ctx.Done():return;case<-t.C:phase+=0.2;set(ns,temp,72+5*math.Sin(phase));set(ns,press,13.5+0.5*math.Sin(phase/2));set(ns,rpm,1800+120*math.Sin(phase/3))}}}();log.Printf("opcua simulator on opc.tcp://%s:%d namespace=%d",*host,*port,ns.ID());<-ctx.Done()}
func set(ns *server.NodeNameSpace,n *server.Node,v float64){dv:=ua.DataValue{Value:ua.MustVariant(v),SourceTimestamp:time.Now(),EncodingMask:ua.DataValueValue|ua.DataValueSourceTimestamp};n.SetAttribute(ua.AttributeIDValue,&dv);ns.ChangeNotification(n.ID())}
