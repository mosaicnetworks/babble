package io.babble.mobile.myapplication;

import android.content.Context;
import android.os.Environment;
import android.system.ErrnoException;
import android.util.Log;

import com.google.gson.Gson;

import java.util.HashMap;
import java.util.Map;
import java.io.*;
import java.lang.Exception;

import mobile.Ball;
import mobile.CommitHandler;
import mobile.ErrorContext;
import mobile.ErrorHandler;
import mobile.EventHandler;
import mobile.Mobile;
import mobile.Node;
import mobile.TxContext;

public class BabbleNode implements CommitHandler, ErrorHandler {

    private Node node;
    private MainActivity context;
    public Map<String, Ball> store = new HashMap<>();
    public ConfigData cnfgData = null;

     ConfigData getConfigData () {

        File folder = context.getExternalFilesDir(null);    //===/storage/sdcard0/Android/data/io.babble.mobile.myapplication/files==
        if (!folder.exists()) {
            throw new ArithmeticException(String.format(String.format("ConfigBabble.json file not found in1 '%s'", folder)));
        }

        File file = new File(folder + "/ConfigBabbleMobile.json");
        if (!file.exists()) {
            throw new  ArithmeticException(String.format("ConfigBabbleMobile.json.json file not found in3 '%s'", file));
        }

        try {
            byte[] fileData = new byte[(int) file.length()];
            DataInputStream dis = new DataInputStream(new FileInputStream(file));
            dis.readFully(fileData);
            dis.close();

            Gson gson = new Gson();
            String strJson = new String(fileData);
            cnfgData = gson.fromJson(strJson, ConfigData.class);
            return cnfgData;
        }catch(Exception e){
            throw new ArithmeticException(String.format("Unexpected error occur while loading json file '%s', %s", file, e.toString()));
        }
    }

    String  Gson2Stirng ( Peer[] pears) {

         try {
            Gson gson = new Gson();
            String strJson = gson.toJson(pears);
            return strJson;
        }catch(Exception e){
            throw new ArithmeticException(String.format("Unexpected error occur while converting json to string. %s",  e.toString()));
        }
    }

    public BabbleNode(Context context) {

        this.context = (MainActivity) context;

        EventHandler events = Mobile.newEventHandler();
        events.onMessage(this);
        events.onError(this);

        cnfgData = getConfigData();

        String nodeAddr = cnfgData.node_addr;
        String privKey =  cnfgData.nodePrivateKey;

        String peers = Gson2Stirng(cnfgData.peers);

        node = Mobile.new_(nodeAddr, peers, privKey, events, Mobile.defaultConfig());
       //node.run();
    }

    public void shutdown() {

        if (node != null)
            node.shutdown();
    }

    public void run() {

        if (node != null)
            node.run(true);
    }

    @Override
    public void onCommit(final TxContext txContext) {

            String nodeId = "N" + txContext.getInfo().getID();
            Ball b = txContext.getBall();
        Log.i("Babble", ">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>.");
            store.put(nodeId, b);
    }

    @Override
    public void onError(final ErrorContext ctx) {

        context.runOnUiThread(new Runnable() {
            @Override
            public void run() {
                Log.i("Babble", "Received OnError event. " + ctx.getError());
                //TextView tv = context.findViewById(R.id.text);
                //tv.setText(ctx.getError());
            }
        });

    }
}
