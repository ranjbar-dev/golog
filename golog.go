package golog

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type GoLog struct {
	ctx         context.Context
	jobsChannel chan Log
	config      Config
	logs        []Log
}

func (l *GoLog) handleLogs() {

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {

		case <-l.ctx.Done():
			fmt.Println("golog context done")
			return

		case <-ticker.C:

			if len(l.logs) > 0 {

				go l.writeServer(l.logs)
				l.logs = make([]Log, 0)
			}

		case record := <-l.jobsChannel:

			if l.config.LogStdout {

				go l.writeStdout(record)
			}

			if l.config.LogFile {

				go l.writeFile(record)
			}

			if l.config.LogServer {

				l.logs = append(l.logs, record)
			}
		}
	}
}

func (l *GoLog) writeStdout(record Log) {

	if record.Data != nil {

		fmt.Printf("%s [%s] %s - %s, data: %v \n", time.Now().Format("2006/01/02 15:04:05"), record.Level.String(), record.Title, record.Message, record.Data)
		return
	} else {

		fmt.Printf("%s [%s] %s - %s \n", time.Now().Format("2006/01/02 15:04:05"), record.Level.String(), record.Title, record.Message)
	}
}

func (l *GoLog) writeFile(record Log) {

	if record.Data != nil {

		log.Printf("[%s] %s - %s, data: %v \n", record.Level.String(), record.Title, record.Message, record.Data)
		return
	} else {

		log.Printf("[%s] %s - %s \n", record.Level.String(), record.Title, record.Message)
	}
}

func (l *GoLog) writeServer(records []Log) {

	var logs []HttpLog
	for _, record := range records {

		logs = append(logs, record.ToHttpLog())
	}

	request := LogsRequest{
		Logs: logs,
	}

	data, err := json.Marshal(request)
	if err != nil {

		fmt.Println("error marshalling logs")
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%s/log/create-many", l.config.ServerApiProtocol, l.config.ServerApiHost, l.config.ServerApiPort), bytes.NewBuffer(data))
	if err != nil {

		fmt.Println("error creating request", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", l.generateHash())
	req.Header.Set("Platform-Name", l.config.ServerPlatfrom)

	// Disable server certificate verification
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {

		fmt.Println("error sending logs, err: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {

			fmt.Println("error reading response body", err)
			return
		}

		fmt.Println("error sending logs status code: ", resp.StatusCode, " body: ", string(body))
		return
	}
}

func (l *GoLog) SetConfig(config Config) {

	l.config = config

	log.SetOutput(&lumberjack.Logger{
		Filename:   config.FileLocation,
		MaxSize:    l.config.FileMaxSize,
		MaxBackups: l.config.FileMaxBackups,
	})
}

func (l *GoLog) Log(level Level, title string, message string, data ...any) {

	go func(level Level, title string, message string, data ...any) {

		l.jobsChannel <- Log{
			Enviroment: l.config.Enviroment,
			Level:      level,
			Title:      title,
			Message:    message,
			Data:       data,
		}
	}(level, title, message, data...)
}

func (l *GoLog) Trace(title string, message string, data ...any) {

	l.Log(Trace, title, message, data...)
}

func (l *GoLog) Debug(title string, message string, data ...any) {

	l.Log(Debug, title, message, data...)
}

func (l *GoLog) Info(title string, message string, data ...any) {

	l.Log(Info, title, message, data...)
}

func (l *GoLog) Warn(title string, message string, data ...any) {

	l.Log(Warn, title, message, data...)
}

func (l *GoLog) Error(title string, message string, data ...any) {

	l.Log(Error, title, message, data...)
}

func (l *GoLog) Fatal(title string, message string, data ...any) {

	l.Log(Fatal, title, message, data...)
}

func (l *GoLog) Panic(title string, message string, data ...any) {

	l.Log(Panic, title, message, data...)
}

func NewGoLog(ctx context.Context, config Config) *GoLog {

	GoLog := &GoLog{
		ctx:         ctx,
		jobsChannel: make(chan Log, 1000),
		config:      config,
		logs:        make([]Log, 0),
	}

	go GoLog.handleLogs()

	return GoLog
}
