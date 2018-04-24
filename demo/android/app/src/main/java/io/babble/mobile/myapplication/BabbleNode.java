package io.babble.mobile.myapplication;

import android.content.Context;
import android.util.Log;

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.util.HashMap;
import java.util.Map;
import java.io.*;

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

    public void SubmitTx(int x, int y, int cR, int cB, int  cF){

        byte[] binBuf = new byte[] {
                (byte)(x & 0x000000FF), (byte)(x & 0x0000FF00), (byte)(x & 0x00FF0000), (byte)(x & 0xFF000000),
                (byte)(y & 0x000000FF), (byte)(y & 0x0000FF00), (byte)(y & 0x00FF0000), (byte)(y & 0xFF000000),
                (byte)(cR & 0x000000FF), (byte)(cR & 0x0000FF00), (byte)(cR & 0x00FF0000), (byte)(cR & 0xFF000000),
                (byte)(cB & 0x000000FF), (byte)(cB & 0x0000FF00), (byte)(cB & 0x00FF0000), (byte)(cB & 0xFF000000),
                (byte)(cF & 0x000000FF), (byte)(cF & 0x0000FF00), (byte)(cF & 0x00FF0000), (byte)(cF & 0xFF000000)};

        node.submitTx(binBuf);
    }

    ConfigData getConfigData () {

        File folder = context.getExternalFilesDir(null);    //===/storage/sdcard0/Android/data/io.babble.mobile.myapplication/files==
        if (!folder.exists()) {
            throw new ArithmeticException(String.format(String.format("ConfigBabble.json file not found in1 '%s'", folder)));
        }

        File file = new File(folder + "/ConfigBabbleMobile.json");
        if (!file.exists()) {
            throw new  ArithmeticException(String.format("getConfigData: ConfigBabbleMobile.json.json file not found in3 '%s'", file));
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
            throw new ArithmeticException(String.format("getConfigData: Unexpected error occur while loading json file '%s', %s", file, e.toString()));
        }
    }

    public void saveConfigData (ConfigData cnfgData) throws IOException {

        File folder = context.getExternalFilesDir(null);    //===/storage/sdcard0/Android/data/io.babble.mobile.myapplication/files==
        if (!folder.exists()) {
            throw new IOException(String.format(String.format("ConfigBabble.json file not found in1 '%s'", folder)));
        }

        File file = new File(folder + "/ConfigBabbleMobile.json");
        if (!file.exists()) {
            throw new IOException (String.format("saveConfigData: ConfigBabbleMobile.json.json file not found in3 '%s'", file));
        }

        try {
                Gson gson = new Gson();
                Type type = new TypeToken<ConfigData>() {}.getType();
                String gsonData = gson.toJson(cnfgData, type);

                Writer output = new BufferedWriter(new FileWriter(file));
                output.write(gsonData);
                output.close();
            } catch (IOException e) {
                e.printStackTrace();
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
