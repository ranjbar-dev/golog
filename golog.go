package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type GoLog struct {
	ctx             context.Context
	jobsChannel     chan Log
	dispatchChannel chan Log
	config          Config
}

func (l *GoLog) handleLogs() {

	var logs []Log

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {

		case <-l.ctx.Done():
			return

		case <-ticker.C:

			if len(logs) > 0 {

				go l.writeServer(logs)
				logs = logs[:0]
			}
			return

		case record := <-l.jobsChannel:

			if l.config.LogStdout {

				go l.writeStdout(record)
			}

			if l.config.LogFile {

				go l.writeFile(record)
			}

			if l.config.LogServer {

				logs = append(logs, record)
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

	client := &http.Client{}
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

	go func() {

		l.jobsChannel <- Log{
			Enviroment: l.config.Enviroment,
			Level:      level,
			Title:      title,
			Message:    message,
			Data:       data,
		}
	}()
}

func NewGoLog(ctx context.Context) *GoLog {

	GoLog := &GoLog{
		ctx:             ctx,
		jobsChannel:     make(chan Log, 1000),
		dispatchChannel: make(chan Log, 1000),
		config:          Config{},
	}

	go GoLog.handleLogs()

	return GoLog
}
