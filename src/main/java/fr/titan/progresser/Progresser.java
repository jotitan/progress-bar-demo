package fr.titan.progresser;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.atomic.AtomicInteger;

public class Progresser {
    private String id;
    private Integer total;
    private AtomicInteger done = new AtomicInteger(0);
    private List<BlockingQueue<String>> queues = new ArrayList<>();

    public Progresser(String id, Integer total){
        this.id = id;
        this.total = total;
    }

    public void done() {
        int currentDone = done.addAndGet(1);
        queues.forEach(q -> {
            try {
                q.put(String.format("{\"total\":%d,\"done\":%d}", total, currentDone));
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        });
    }

    public void end(){
        queues.forEach(q -> {
            try {
                q.put("");
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        });
    }

    synchronized
    public void add(BlockingQueue q){
        queues.add(q);
    }

    public String getId() {
        return id;
    }
}
