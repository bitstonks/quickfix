// Copyright (c) quickfixengine.org  All rights reserved.
//
// This file may be distributed under the terms of the quickfixengine.org
// license as defined by quickfixengine.org and appearing in the file
// LICENSE included in the packaging of this file.
//
// This file is provided AS IS with NO WARRANTY OF ANY KIND, INCLUDING
// THE WARRANTY OF DESIGN, MERCHANTABILITY AND FITNESS FOR A
// PARTICULAR PURPOSE.
//
// See http://www.quickfixengine.org/LICENSE for licensing information.
//
// Contact ask@quickfixengine.org if any conditions of this licensing
// are not clear to you.

package quickfix

import (
	"bytes"
	"io"
	"time"
)

func writeLoop(connection io.Writer, messageOut chan []byte, log Log) {
	for {
		msg, ok := <-messageOut
		if !ok {
			return
		}

		if _, err := connection.Write(msg); err != nil {
			log.OnEvent(err.Error())
		}
	}
}

func batchWriteLoop(connection io.Writer, messageOut chan []byte, log Log) {
	maxBatchSize := 100
	maxBatchDuration := 25 * time.Millisecond
	tick := time.NewTicker(maxBatchDuration)
	newline := []byte("\n")

	for {
		tick.Reset(maxBatchDuration)
		messages := make([][]byte, 0, maxBatchSize)
	innerLoop:

		for {
			select {
			case msg, ok := <-messageOut:
				if !ok {
					break innerLoop
				}

				messages = append(messages, msg)

				if len(messages) >= maxBatchSize {
					break innerLoop
				}

			case <-tick.C:
				break innerLoop
			}
		}

		if len(messages) > 0 {
			if _, err := connection.Write(bytes.Join(messages, newline)); err != nil {
				log.OnEvent(err.Error())
			}
		}
	}
}

func readLoop(parser *parser, msgIn chan fixIn, log Log) {
	defer close(msgIn)

	for {
		msg, err := parser.ReadMessage()
		if err != nil {
			log.OnEvent(err.Error())
			return
		}
		msgIn <- fixIn{msg, parser.lastRead}
	}
}
