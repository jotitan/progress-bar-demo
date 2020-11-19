package fr.titan.progresser;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.StreamingResponseBody;
import reactor.util.concurrent.Queues;

import javax.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.io.OutputStream;
import java.util.Queue;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.IntStream;

@RestController
public class ProgressController {

    private static final Logger LOG = LoggerFactory.getLogger(ProgressController.class);

    private AtomicInteger counter = new AtomicInteger(0);


    private ConcurrentHashMap<String,Progresser> progressers = new ConcurrentHashMap();

    @GetMapping(value = "/launch",produces = MediaType.APPLICATION_JSON_VALUE )
    public String launchTasks(@RequestParam("nb") Integer nb,
                              @RequestParam("time") Integer time,
                              @RequestParam("thread") Integer thread,
                              HttpServletResponse response) {

        String id = String.format("%d",counter.addAndGet(1));
        Progresser progresser = new Progresser(id, nb);
        progressers.put(id, progresser);
        new Thread(()->runTasks(nb,time,thread,progresser)).start();
        response.setHeader("Access-Control-Allow-Origin", "*");

        return String.format("{\"id\":\"%s\"}",id);
    }

    @GetMapping(value = "/listen" )
    public ResponseEntity<StreamingResponseBody> listenTasks(@RequestParam("id")String id, HttpServletResponse response){
        response.setHeader("Content-Type", "text/event-stream");
        response.setHeader("Cache-Control", "no-cache");
        response.setHeader("Connection", "keep-alive");
        response.setHeader("Access-Control-Allow-Origin", "*");

        BlockingQueue queue = new LinkedBlockingQueue<>();
        Progresser progresser = progressers.get(id);
        if(progresser == null){return null;}
        progresser.add(queue);

        StreamingResponseBody stream = out -> {
            while(true) {
                try {
                    String message = (String)queue.take();
                    if("".equals(message)){
                        writeMessage(out,"end","{\"end\":true}");
                        break;
                    }
                    writeMessage(out,"stat",message);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }
        };
        return new ResponseEntity<>(stream, HttpStatus.OK);
    }

    private void writeMessage(OutputStream out, String event, String message){
        try{
            out.write(String.format("event: %s%n",event).getBytes());
            out.write(String.format("data: %s%n%n",message).getBytes());
            out.flush();
        }catch(IOException ioex){
            ioex.printStackTrace();
        }
    }

    private void runTasks(int nb, int time, int thread,Progresser progresser){
        ExecutorService executor = Executors.newFixedThreadPool(thread);
        IntStream.range(0,nb).forEach(i->executor.submit(()->runTask(time,progresser)));
        executor.shutdown();
        try {
            executor.awaitTermination(10,TimeUnit.MINUTES);
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
        LOG.info(String.format("End of tasks %s",progresser.getId()));
        progresser.end();
    }

    private void runTask(int time,Progresser progresser){
        try {
            Thread.sleep(time);
            progresser.done();
            LOG.info(String.format("End task %s",progresser.getId()));
        } catch (InterruptedException e) {
            e.printStackTrace();
        }
    }


}
