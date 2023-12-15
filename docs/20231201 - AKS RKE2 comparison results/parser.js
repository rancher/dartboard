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

function convertToMs(val) {
  if (val.endsWith("ms")) {
    return val.slice(0,-2)
  }
  
  if (val.endsWith("s")) {
    val = val.slice(0,-1)
    return Number(val) * 1000
  }
}

// 0d00h50m21.0s
function convertTimeToSeconds(val) {
  let totSec = 0
  data = (val.trim()).split("d")
  data.shift()
  data = data[0].split("h")
  totSec += Number(data[0])*3600
  data.shift()
  data = data[0].split("m")
  totSec += Number(data[0])*60
  data.shift()
  totSec += Number(data[0].slice(0, -1))
  return totSec
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

      var fs_idx = fs.openSync(outFileName, "a+")
      fs.writeSync(fs_idx, fileName + ",")
      for (j in data) {
        // avg, min, med, max, P(90), P(95)
        let pair = data[j].split("=")

        // be sure all values are in ms and remove trailing unit suffix
        let val = convertToMs(pair[1])

        fs.writeSync(fs_idx, val + ",")
      }

      // Extract total duration from last line, e.g.:
      // list ✓ [ 100% ] 1 VUs  0d00h50m21.0s/24h0m0s  30/30 iters, 30 per VU
      //                             ^^
      var line = lines[lines.length - 2];
      console.log(line) 
      data = line.split(/\s+/)
      for (i=0; i < 7; i++) {
        data.shift()
      }
      fs.writeSync(fs_idx, "," + convertTimeToSeconds(data[0].split("/")[0]) + "\n")

      fs.closeSync(fs_idx)
      break
    }
  }


} catch (err) {
  console.error("Cannot read file " + fileName + "\n" + err)
}