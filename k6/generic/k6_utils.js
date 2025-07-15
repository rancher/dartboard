
export function sizeOfHeaders(headers) {
  return Object.keys(headers).reduce((sum, key) => sum + key.length + headers[key].length, 0);
}

export function trackResponseSizePerURL(res, tags, headerDataRecv, epDataRecv) {
  // Add data points for received data
  headerDataRecv.add(sizeOfHeaders(res.headers));
  if (res.hasOwnProperty('body') && res.body) {
    epDataRecv.add(res.body.length, tags);
  } else {
    epDataRecv.add(0, tags);
  }
}
