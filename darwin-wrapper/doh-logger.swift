#!/usr/bin/swift

/*
   DNS-over-HTTPS
   Copyright (C) 2017-2018 Star Brilliant <m13253@hotmail.com>

   Permission is hereby granted, free of charge, to any person obtaining a
   copy of this software and associated documentation files (the "Software"),
   to deal in the Software without restriction, including without limitation
   the rights to use, copy, modify, merge, publish, distribute, sublicense,
   and/or sell copies of the Software, and to permit persons to whom the
   Software is furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in
   all copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
   FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
   DEALINGS IN THE SOFTWARE.
*/

import Foundation
import os.log

if CommandLine.arguments.count < 3 {
    let programName = CommandLine.arguments[0]
    print("Usage: \(programName) LOG_NAME PROGRAM [ARGUMENTS]\n")
    exit(1)
}
let logSubsystem = CommandLine.arguments[1]
let logger = OSLog(subsystem: logSubsystem, category: "default")

let pipe = Pipe()
var buffer = Data()
NotificationCenter.default.addObserver(forName: FileHandle.readCompletionNotification, object: pipe.fileHandleForReading, queue: nil) { notification in
    let data = notification.userInfo?["NSFileHandleNotificationDataItem"] as? Data ?? Data()
    buffer.append(data)
    var lastIndex = 0
    for (i, byte) in buffer.enumerated() {
        if byte == 0x0a {
            let line = String(data: buffer.subdata(in: lastIndex..<i), encoding: .utf8) ?? ""
            print(line)
            os_log("%{public}@", log: logger, line)
            lastIndex = i + 1
        }
    }
    buffer = buffer.subdata(in: lastIndex..<buffer.count)
    if data.count == 0 && buffer.count != 0 {
        let line = String(data: buffer, encoding: .utf8) ?? ""
        print(line, terminator: "")
        os_log("%{public}@", log: logger, line)
    }
    pipe.fileHandleForReading.readInBackgroundAndNotify()
}
pipe.fileHandleForReading.readInBackgroundAndNotify()

let process = Process()
process.arguments = Array(CommandLine.arguments[3...])
process.executableURL = URL(fileURLWithPath: CommandLine.arguments[2])
process.standardError = pipe.fileHandleForWriting
process.standardInput = FileHandle.standardInput
process.standardOutput = pipe.fileHandleForWriting
NotificationCenter.default.addObserver(forName: Process.didTerminateNotification, object: process, queue: nil) { notification in
    if buffer.count != 0 {
        let line = String(data: buffer, encoding: .utf8) ?? ""
        print(line, terminator: "")
        os_log("%{public}@", log: logger, line)
    }
    exit(process.terminationStatus)
}

let SIGINTSource = DispatchSource.makeSignalSource(signal: SIGINT)
let SIGTERMSource = DispatchSource.makeSignalSource(signal: SIGTERM)
SIGINTSource.setEventHandler(handler: process.interrupt)
SIGTERMSource.setEventHandler(handler: process.terminate)
signal(SIGINT, SIG_IGN)
signal(SIGTERM, SIG_IGN)
SIGINTSource.resume()
SIGTERMSource.resume()

do {
    try process.run()
} catch {
    let errorMessage = error.localizedDescription
    print(errorMessage)
    os_log("%{public}@", log: logger, type: .fault, errorMessage)
    exit(1)
}

RunLoop.current.run()
