import logo from './logo.svg';
import './App.css';
import Webserver from './webSocketService';
import { useEffect, useState } from 'react';
import Card from './component/ui-card';

function App() {

  // var socket = new WebSocket("");
  // var socket = new WebSocket("wss://demo.piesocket.com/v3/channel_123?api_key=VCXCEuvhGcBDP7XhiJJUDvR1e1D3eiVjgZ9VRiaV&notify_self");

const[msg,setMsg] = useState();


// socket.onclose = function() {
//   console.log("WebSocket connection closed");
// };

useEffect(()=>{
  let ws = new WebSocket('ws://0.tcp.in.ngrok.io:17856/dashboard');
  ws.onopen = () => console.log('ws opened');
  ws.onclose = () => console.log('ws closed');

  ws.onmessage = e => {
    setMsg(Object.entries(JSON.parse(e.data)));
  };

  return () => {
    ws.close();
  }
},[])

  return (
    <div className="App">
      <header className="App-header">
     <h1>Dashboard</h1>
     </header>
      <main>
        {msg &&
        msg.map((data)=>{
         return <Card lable = {data[0]} data = {data[1]}/>
        })
        }
    </main>
    </div>
  );
}

export default App;
