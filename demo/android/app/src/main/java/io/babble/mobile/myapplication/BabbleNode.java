package io.babble.mobile.myapplication;

import android.content.Context;
import android.media.MediaScannerConnection;
import android.widget.Toast;

import com.google.gson.Gson;
import com.google.gson.reflect.TypeToken;

import java.lang.reflect.Type;
import java.net.Inet4Address;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.net.SocketException;
import java.util.Arrays;
import java.util.Enumeration;
import java.util.HashMap;
import java.util.Map;
import java.io.*;
import java.util.Random;

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

    public long transCount = 0;               //we'll count transactions here
    public Map<String, Ball> store = new HashMap<>();
    public ConfigData cnfgData = null;

    public BabbleNode(Context context) {

        this.context = (MainActivity) context;

        EventHandler events = Mobile.newEventHandler();
        events.onMessage(this);
        events.onError(this);

        Mobile.getPrivPublKeys();

        cnfgData = getConfigData();

        if ( (cnfgData == null) || (cnfgData.nodePrivateKey == null) || ( cnfgData.nodePrivateKey.isEmpty()) ){
            createNodeCert();
        }

        String nodeAddr = cnfgData.node_addr;
        String privKey =  cnfgData.nodePrivateKey;

        String peers = Gson2Stirng(cnfgData.peers);

        node = Mobile.new_(nodeAddr, peers, privKey, events, Mobile.defaultConfig());

       //node.run();
    }

    public void shutdown() {

        if ( node != null )
            node.shutdown();
    }

    public void run() {

        if ( node != null ) {
            node.run(true);
        }
    }

    @Override
    public void onCommit(final TxContext txContext) {

        String nodeId = "N" + txContext.getInfo().getID();
        Ball b = txContext.getBall();

        transCount++;
        store.put(nodeId, b);
    }

    @Override
    public void onError(final ErrorContext ctx) {

        context.runOnUiThread(new Runnable() {
            @Override
            public void run() {
                Toast.makeText(context, "BabbleNode.onError.run '" +  ctx.getError() + "'.", Toast.LENGTH_LONG).show();
            }
        });
    }

    ConfigData getConfigData () {

        File folder = context.getExternalFilesDir(null);    //===/storage/sdcard0/Android/data/io.babble.mobile.myapplication/files==
        if ( !folder.exists() ) {
            Toast.makeText(context, "getConfigData:\r\nThe config folder - not found.\r\n'" + folder + "'.", Toast.LENGTH_LONG).show();
            return new ConfigData();
        }

        File file = new File(folder + "/ConfigBabbleMobile.json");
        if ( !file.exists() ) {
            Toast.makeText(context, "getConfigData: Config file not found.\r\nIt will be generated automatically.\r\nFile: '" + file + "'.", Toast.LENGTH_LONG).show();
            return new ConfigData();
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
            Toast.makeText(context, String.format("getConfigData:\r\nAn unexpected error occurred while loading json file.\r\nFile: '%s',\r\nError: %s", file, e.toString()), Toast.LENGTH_LONG).show();
            return new ConfigData();
        }
    }

    public void saveConfigData (ConfigData cnfgData) throws IOException {

        File folder = context.getExternalFilesDir(null);    //===/storage/sdcard0/Android/data/io.babble.mobile.myapplication/files==
        if ( !folder.exists() ) {
            Toast.makeText(context, String.format("saveConfigData:\r\nThe config folder not found\r\nFolder: '%s'.", folder), Toast.LENGTH_LONG).show();
            return;
        }

        File file = new File(folder,  "/ConfigBabbleMobile.json");
        if ( !file.exists() ) {   //The file will be created
            file.createNewFile();

            BufferedWriter writer = new BufferedWriter(new FileWriter(file, true));
            writer.write("");
            writer.close();

            MediaScannerConnection.scanFile(context, new String[]{file.toString()}, null,null);  //the file to be seen by us
        }

        try {
            Gson gson = new Gson();
            Type type = new TypeToken<ConfigData>() {}.getType();
            String gsonData = gson.toJson(cnfgData, type);

            Writer output = new BufferedWriter(new FileWriter(file));
            output.write(gsonData);
            output.close();
        } catch (IOException e) {
            Toast.makeText(context, String.format(String.format("saveConfigData:\r\nAn unexpected error occurred while saving config data\r\nFile: '%s'.\r\nError: %s", folder, e.toString())), Toast.LENGTH_LONG).show();
        }
    }

    String  Gson2Stirng ( Peer[] pears) {
        try {
            Gson gson = new Gson();
            return gson.toJson(pears);
        }catch(Exception e){
            Toast.makeText(context, String.format(String.format("saveConfigData:\r\nAn unexpected error occurred while serializing json object\r\nError: '%s'", e.toString())), Toast.LENGTH_LONG).show();
            return "";
        }
    }

    public String getLocalIpAddress() {
        try {
            for (Enumeration<NetworkInterface> en = NetworkInterface.getNetworkInterfaces(); en.hasMoreElements();) {
                NetworkInterface intf = en.nextElement();
                for (Enumeration<InetAddress> enumIpAddr = intf.getInetAddresses(); enumIpAddr.hasMoreElements();) {
                    InetAddress inetAddress = enumIpAddr.nextElement();
                    if (!inetAddress.isLoopbackAddress() && inetAddress instanceof Inet4Address) {
                        return inetAddress.getHostAddress();
                    }
                }
            }
        } catch (SocketException e) {
            Toast.makeText(context, "getLocalIpAddress:\r\nAn unexpected error.\r\nError: '" + e.toString() + "'.", Toast.LENGTH_LONG).show();
        }
        return "";
    }

    public  boolean saveNodeData(String privateKey, String publicKey, String ip) { //each string is netAddr*pubKeyHex*active

        String[] arrColor = {"BLACK", "BLUE", "CYAN", "DKGRAY", "GRAY", "GREEN", "LTGRAY", "MAGENTA", "RED", "WHITE", "YELLOW"};  //base color names in android by default
        Random rndInt = new Random();

        try{
            if ( cnfgData == null ){
                cnfgData = new ConfigData();
            }

            if ( (cnfgData.nodePrivateKey == null) || cnfgData.nodePrivateKey.isEmpty() ){  //not exists
                cnfgData.nodeID = 1;
                cnfgData.nodePrivateKey = privateKey;
                cnfgData.nodePublicKey = publicKey;
                cnfgData.node_addr = ip + ":1331";
                cnfgData.proxy_addr = ip + ":1332";
                cnfgData.client_addr = ip + ":1333";
                cnfgData.service_addr = ip + ":1334";
                cnfgData.storeType = "inmem";
                cnfgData.storePath = "";

                int i1 = rndInt.nextInt(10);    //in [0, 10]
                int i2 = rndInt.nextInt(10);
                while (i1 == i2){
                    i2 = rndInt.nextInt(10);
                }

                cnfgData.circleBackColor = arrColor[i1];
                cnfgData.circleForeColor = arrColor[i2];

                i1 = rndInt.nextInt(100) + 50;    //in [50, 150]
                cnfgData.circleRadius = i1;

                if ( cnfgData.peers == null ){
                    cnfgData.peers = new Peer[1];
                    cnfgData.peers[0] = new Peer();
                }else{
                    cnfgData.peers = Arrays.copyOf(cnfgData.peers, cnfgData.peers.length + 1);
                }

                i1 = cnfgData.peers.length - 1;
                cnfgData.peers[i1].netAddr = cnfgData.node_addr;
                cnfgData.peers[i1].pubKeyHex = cnfgData.nodePublicKey;
                cnfgData.peers[i1].nickName = "Your NickName";
                cnfgData.peers[i1].active = 1;
            }

            saveConfigData(cnfgData);
            return true;
        }catch (Exception e){
            Toast.makeText(context, "saveNodeData: An unexpected error '" + e.toString() + "'.", Toast.LENGTH_LONG).show();
            return false;
        }
    }

    private void createNodeCert(){
        String keyPair = Mobile.getPrivPublKeys();    //publicKey[!@#$%^]privateKey
        String[] separated =  keyPair.split("=!@#@!=");
        String publicKey = separated[0].trim();
        String privateKey = separated[1].trim();

        String ip = getLocalIpAddress();
        saveNodeData(privateKey, publicKey, ip);
    }
}
