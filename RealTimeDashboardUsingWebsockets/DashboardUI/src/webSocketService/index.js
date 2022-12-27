import { useEffect, useMemo, useState } from 'react';

const Webserver = ()=>
{

  var socket = new WebSocket("wss://demo.piesocket.com/v3/channel_123?api_key=VCXCEuvhGcBDP7XhiJJUDvR1e1D3eiVjgZ9VRiaV&notify_self");

const[msg,setMsg] = useState("");


useEffect(()=>{
  socket.onopen = function() {
  console.log("WebSocket connection opened");
  socket.send("Hello, server!");
}
  console.log(socket)
  socket.onmessage = function(event) {
  console.log("Received message: " + event.data);
  setMsg(event.data)
};

},[socket])

const massage = useMemo(()=>{
    return msg;
},[msg,setMsg])
massage();

}
export default Webserver;