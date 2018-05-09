package io.babble.mobile.myapplication;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.DialogInterface;
import android.content.Intent;
import android.os.Bundle;
import android.view.KeyEvent;
import android.widget.ListView;
import android.widget.Toast;

import com.google.android.gms.common.api.CommonStatusCodes;

import java.math.BigInteger;
import java.nio.charset.Charset;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;

public class ListViewDialog extends Activity{

    private String statPeersHash = "";

    private AdapterListView adapterForChecks;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_listview);

        ArrayList<String> rawPeerData = getIntent().getStringArrayListExtra("RawNodeSettings");
        ArrayList<ListViewItem> lvPeerData =  new ArrayList<ListViewItem>();

        String allRawPeerData = "";

        for (int i = 0; i < rawPeerData.size(); i++){
            String[] separated = rawPeerData.get(i).split("#!D#@!");
            allRawPeerData = allRawPeerData + rawPeerData.get(i) + "!!!#!D#@!!!!";

            lvPeerData.add(new ListViewItem (separated[0], separated[1], separated[2].contains("1")));
        }

        statPeersHash = md5(allRawPeerData);

        ListView listView = (ListView) findViewById(R.id.listview);
        AdapterListView adapter = new AdapterListView(this, R.layout.item_listview, lvPeerData);
        adapterForChecks = adapter;
        listView.setAdapter(adapter);
    }
    private String md5(String s)
    {
        MessageDigest digest;
        try
        {
            digest = MessageDigest.getInstance("MD5");
            digest.update(s.getBytes(Charset.forName("US-ASCII")),0,s.length());
            byte[] magnitude = digest.digest();
            BigInteger bi = new BigInteger(1, magnitude);
            String hash = String.format("%0" + (magnitude.length << 1) + "x", bi);
            return hash;
        }
        catch (NoSuchAlgorithmException e)
        {
            e.printStackTrace();
        }
        return "";
    }

    public void onBackPressed(){

        String allRawPeerData = "";
        for (int i = 0; i < adapterForChecks.getCount(); i++){
            allRawPeerData = allRawPeerData + adapterForChecks.arrayPeers.get(i).nodeAddr + "#!D#@!";
            allRawPeerData = allRawPeerData + adapterForChecks.arrayPeers.get(i).nickName + "#!D#@!";
            allRawPeerData = allRawPeerData + (adapterForChecks.arrayPeers.get(i).active ? "1" : "0") + "!!!#!D#@!!!!";
        }
        String statPeersHashLoc = md5(allRawPeerData);

        if (statPeersHashLoc.contentEquals(statPeersHash)){
            ListViewDialog.super.onBackPressed();
            return;
        }

        AlertDialog.Builder dlgAlert  = new AlertDialog.Builder(this);
        dlgAlert.setMessage("You have changes in the data.\r\nDo you want to save them?");
        dlgAlert.setTitle("Babble.");
        dlgAlert.setPositiveButton("Yes", new DialogInterface.OnClickListener() {
                                                   public void onClick(DialogInterface dialog, int which) {
                                                       ArrayList<String> rawNewPeerData = new ArrayList<String> ();
                                                       for (int i = 0; i < adapterForChecks.getCount(); i++){
                                                           String onePeer = adapterForChecks.arrayPeers.get(i).nodeAddr + "#!D#@!";
                                                           onePeer = onePeer + adapterForChecks.arrayPeers.get(i).nickName + "#!D#@!";
                                                           onePeer =  onePeer + (adapterForChecks.arrayPeers.get(i).active ? "1" : "0");

                                                           rawNewPeerData.add(onePeer);
                                                       }

                                                       Intent data = new Intent();
                                                       data.putStringArrayListExtra("rawPeerData", rawNewPeerData);
                                                       setResult(CommonStatusCodes.SUCCESS, data);
                                                       ListViewDialog.super.onBackPressed();
                                                }});
        dlgAlert.setNegativeButton("No", new DialogInterface.OnClickListener() {
                                                public void onClick(DialogInterface dialog, int which) {
                                                    ListViewDialog.super.onBackPressed();
                                                }});
        dlgAlert.create().show();
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        switch(keyCode){
            case KeyEvent.KEYCODE_BACK:
                onBackPressed();
                return true;
        }
        return super.onKeyDown(keyCode, event);
    }
}
