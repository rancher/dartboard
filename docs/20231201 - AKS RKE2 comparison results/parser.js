#!/usr/bin/env node
var fs = require('fs')
const { argv } = require('process')

if (! argv[2]) {
  console.log("pass filename to parse")
  process.exit(1)
}

var fileName = argv[2]
var outFileName = "output.csv"
var metric = "http_req_duration"

if (argv[3]) {
  outFileName = argv[3]
}
if (argv[4]) {
  metric = argv[4]
}

try {
  var lines = fs.readFileSync(fileName, 'utf8').split("\n")
  for (i in lines) {
    var idx = lines[i].search(metric)

    // found the searched metric
    if (idx != -1) {
      var line = lines[i].trim()
      console.log("found metric '" + metric + "' on line " + i + "\n"  + line)
      data = line.split(/\s+/)

      // drop metric name
      data.shift()

      //fs.writeFileSync("output.txt", "ciao", {flag: "a+"})
      var fs_idx = fs.openSync(outFileName, "a+")
      fs.writeSync(fs_idx, fileName + ",")
      for (j in data) {
        var sep = ","
        if ( data.length == Number(j) +1) {
          sep = "\n"
        }
        var pair = data[j].split("=")
        fs.writeSync(fs_idx, pair[1] + sep)
      }
      fs.closeSync(fs_idx)

    }
  }
} catch (err) {
  console.error("Cannot read file " + fileName + "\n" + err)
}