package main

import (
    "bytes"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/gopcua/opcua"
    "github.com/gopcua/opcua/ua"
)

type nodesFlag []string
func (n *nodesFlag) String() string { return strings.Join(*n, ",") }
func (n *nodesFlag) Set(v string) error { *n=append(*n,v); return nil }

type sample struct { Protocol string `json:"protocol"`; Endpoint string `json:"endpoint"`; Values map[string]any `json:"values"`; TS time.Time `json:"ts"` }

func main(){
    endpoint:=flag.String("endpoint","opc.tcp://127.0.0.1:4840","OPC-UA endpoint")
    edge:=flag.String("edge","http://127.0.0.1:8080","relay-edge URL")
    device:=flag.String("device","opcua-01","device id")
    interval:=flag.Duration("interval",2*time.Second,"poll interval")
    var nodes nodesFlag;flag.Var(&nodes,"node","name=nodeid, repeatable")
    flag.Parse();if len(nodes)==0{nodes=[]string{"temperature=ns=2;s=Temperature","pressure=ns=2;s=Pressure"}}
    ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer cancel()
    c,err:=opcua.NewClient(*endpoint,opcua.SecurityPolicy("None"),opcua.SecurityMode(ua.MessageSecurityModeNone),opcua.AuthAnonymous());if err!=nil{log.Fatal(err)}
    if err=c.Connect(ctx);err!=nil{log.Fatal(err)};defer c.Close(context.Background())
    ids:=map[string]*ua.NodeID{};for _,x:=range nodes{p:=strings.SplitN(x,"=",2);if len(p)!=2{log.Fatalf("bad -node %q",x)};id,err:=ua.ParseNodeID(p[1]);if err!=nil{log.Fatal(err)};ids[p[0]]=id}
    t:=time.NewTicker(*interval);defer t.Stop();hc:=&http.Client{Timeout:3*time.Second}
    for{select{case<-ctx.Done():return;case<-t.C:vals:=map[string]any{};for name,id:=range ids{v,err:=c.Node(id).Value(ctx);if err!=nil{log.Printf("read %s: %v",name,err);continue};vals[name]=v.Value()};body,_:=json.Marshal(sample{Protocol:"opcua",Endpoint:*endpoint,Values:vals,TS:time.Now().UTC()});url:=strings.TrimRight(*edge,"/")+"/v1/ingest/"+*device;req,_:=http.NewRequestWithContext(ctx,http.MethodPost,url,bytes.NewReader(body));req.Header.Set("Content-Type","application/json");resp,err:=hc.Do(req);if err!=nil{log.Printf("ingest: %v",err);continue};resp.Body.Close();if resp.StatusCode/100!=2{log.Printf("ingest status %s",resp.Status)}}}
}
var _=fmt.Sprintf
