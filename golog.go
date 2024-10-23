package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

	for {
		select {

		case <-l.ctx.Done():
			return

		case <-l.dispatchChannel:

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

			if l.config.LogFile {

				logs = append(logs, record)
			}
		}
	}
}

func (l *GoLog) writeStdout(record Log) {

	fmt.Printf("[%s] %s - %s", record.Level.String(), record.Title, record.Message)
}

func (l *GoLog) writeFile(record Log) {

	log.Printf("[%s] %s - %s", record.Level.String(), record.Title, record.Message)
}

func (l *GoLog) writeServer(records []Log) {

	data, err := json.Marshal(records)
	if err != nil {

		fmt.Println("error marshalling logs")
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s://%s:%s", l.config.ServerApiProtocol, l.config.ServerApiHost, l.config.ServerApiPort), bytes.NewBuffer(data))
	if err != nil {

		fmt.Println("error creating request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", l.generateHash())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {

		fmt.Println("error sending logs, err: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		fmt.Println("error sending logs status code: ", resp.StatusCode)
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

func (l *GoLog) Log(record Log) {

	go func() {

		l.jobsChannel <- record
	}()
}

func NewGoLog(ctx context.Context) *GoLog {

	GoLog := &GoLog{
		ctx:             ctx,
		jobsChannel:     make(chan Log, 1000),
		dispatchChannel: make(chan Log, 1000),
		config:          Config{},
	}

	GoLog.handleLogs()

	return GoLog
}
