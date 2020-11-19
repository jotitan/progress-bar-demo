package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)


func main(){
	RunServer()
}

var progressers = make(map[string]*progress)

type stat struct {
	done int
	total int
}

type sseWatch struct {
	w http.ResponseWriter
	// Receive message to be written
	chanel chan stat
}

func (sse sseWatch)runWatch(){
	sse.w.Header().Set("Content-Type", "text/event-stream")
	sse.w.Header().Set("Cache-Control", "no-cache")
	sse.w.Header().Set("Connection", "keep-alive")
	sse.w.Header().Set("Access-Control-Allow-Origin", "*")
	// Wait new messages and send to sse
	for {
		if s,more := <- sse.chanel ; more {
			sse.writeEvent(s)
		} else{
			log.Println("end")
		}
	}
}

func (sse sseWatch)writeEvent(st stat){
	sse.w.Write([]byte("event: stat\n"))
	sse.w.Write([]byte(fmt.Sprintf("data: {\"done\":%d,\"total\":%d}\n\n",st.done,st.total)))
	sse.w.(http.Flusher).Flush()
}

func (sse sseWatch) end() {
	sse.w.Write([]byte("event: end\n"))
	sse.w.Write([]byte("data: {\"end\":true}\n\n"))
	sse.w.(http.Flusher).Flush()
}

func newSse(w http.ResponseWriter,r * http.Request)*sseWatch{
	return &sseWatch{w,make(chan stat)}
}

type progress struct {
	id string
	// Number total of tasks
	total int
	totalDone int
	// Watchers
	sses []*sseWatch
	// thread safe chanel to receive progression
	chanel chan struct{}
}

func newProgress(id string,nb int)*progress{
	progresser := &progress{id:id,total:nb,sses:make([]*sseWatch,0),chanel:make(chan struct{})}
	go progresser.runWatch()
	return progresser
}

func (p * progress)runWatch(){
	for {
		if _,more := <- p.chanel; more {
			p.totalDone++
			p.sendMessage(stat{done:p.totalDone,total:p.total})
		}else{
			for _,s := range p.sses{
				s.end()
			}
			break
		}
	}
}

func (p * progress)sendMessage(message stat){
	for _,s := range p.sses{
		s.writeEvent(message)
	}
}

func (p * progress) done() {
	p.chanel <- struct{}{}
}

func (p * progress)end(){
	close(p.chanel)
}

func addSSE(id string,w http.ResponseWriter,r *http.Request)(*sseWatch,error){
	if up,ok := progressers[id] ; ok {
		sse := newSse(w,r)
		up.sses = append(up.sses,sse)
		return sse,nil
	}
	return nil,errors.New("unknown upload id " + id + " for SSE (" + fmt.Sprintf("%d",len(progressers)))
}

func listenTaskProgress(w http.ResponseWriter,r * http.Request){
	id := r.FormValue("id")
	if sse,err := addSSE(id,w,r) ; err == nil {
		// Block to write messages, otherwise, connexion end
		sse.runWatch()
		log.Println("End watch")
	}else{
		log.Println("Impossible to watch upload",err.Error())
		http.Error(w,err.Error(),404)
	}
}

func runTask(timeInMs int,progresser *	progress){
	time.Sleep(time.Duration(timeInMs)*time.Millisecond)
	log.Println("End task")
	progresser.done()
}

func runTasks(nb,timeInMs,thread int, id string){
	progresser := newProgress(id,nb)
	progressers[id] = progresser
	log.Println("Run tasks",id,":",nb,timeInMs)
	limiter := make(chan struct{},thread)
	waiter := sync.WaitGroup{}
	waiter.Add(nb)
	for i := 0 ; i < nb ; i++ {
		go func(){
			limiter <- struct{}{}
			runTask(timeInMs,progresser)
			<-limiter
			waiter.Done()
		}()
	}
	waiter.Wait()
	progresser.end()
	log.Println("End of tasks",id)
}

func launchTask(w http.ResponseWriter,r * http.Request){
	nb,timeInMs,nbThreads := getParameters(r)
	id := generateUniqueId()
	go runTasks(nb,timeInMs,nbThreads,id)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"id\":\"%s\"}",id)))
}

func getParameters(r * http.Request)(int,int,int){
	return getValue(r.FormValue("nb"),1),
	getValue(r.FormValue("time"),1000),
	getValue(r.FormValue("thread"),1)
}

var counter = int32(0)
func generateUniqueId()string{
	id :=  int(atomic.AddInt32(&counter,1))
	return fmt.Sprintf("%d",id)
}

func getValue(name string,defaultValue int)int{
	if value,err := strconv.ParseInt(name,10,32) ; err == nil {
		return int(value)
	}
	return defaultValue
}

func RunServer(){

	server := http.ServeMux{}
	server.HandleFunc("/launch",launchTask)
	server.HandleFunc("/listen",listenTaskProgress)

	log.Println("Run server test")
	log.Fatal(http.ListenAndServe(":9004",&server))
}
