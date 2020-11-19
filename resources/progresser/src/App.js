import React,{useState} from 'react';
import './App.css';
import axios from "axios";
import 'antd/dist/antd.css';
import {Progress} from "antd";

const backendUrl = "http://localhost:9004";

function App() {
  const [nb,setNb] = useState(1);
  const [time,setTime] = useState(1000);
  const [thread,setThread] = useState(1 );
  const [percent,setPercent] = useState(0);
  const [begin,setBegin] = useState(new Date());
  const [currentDate,setCurrentDate] = useState(new Date());

  const launch = ()=>{
      setPercent(0);
      setBegin(new Date());
      setCurrentDate(new Date());
      axios.get(`${backendUrl}/launch?nb=${nb}&time=${time}&thread=${thread}`)
          .then(data=>{
              let es = new EventSource(`${backendUrl}/listen?id=${data.data.id}`)
              es.addEventListener('stat',data=>{
                  let message = JSON.parse(data.data);
                  setCurrentDate(new Date());
                  setPercent(Math.round((message.done / message.total)*100));
              });
              es.addEventListener('end',()=>{
                  es.close();
                  setPercent(100)
              });
          })
  }

  return (
    <div className="App">
      <header >


      <div>
        Nb : <input value={nb} type={"text"} onChange={value=>setNb(value.target.value)}/><br/>
        Time : <input value={time} type={"text"} onChange={value=>setTime(value.target.value)}/><br/>
        Thread : <input value={thread} type={"text"} onChange={value=>setThread(value.target.value)}/><br/>

        <button onClick={launch}>Launch</button>
         <div>
             <Progress percent={percent} style={{width:300,margin:'auto'}}/>
         </div>
          <div>
              Duration : {Math.round((currentDate - begin)/1000)}s
          </div>

      </div>
      </header>
    </div>
  );
}

export default App;
