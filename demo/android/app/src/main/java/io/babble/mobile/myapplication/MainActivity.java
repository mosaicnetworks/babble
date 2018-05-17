package io.babble.mobile.myapplication;

import android.app.AlertDialog;
import android.content.DialogInterface;
import android.content.Intent;
import android.graphics.Color;
import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;

import android.view.KeyEvent;
import android.view.Menu;
import android.view.MenuItem;

import android.widget.Toast;

import com.google.android.gms.common.api.CommonStatusCodes;

import java.util.ArrayList;

import mobile.Node;

public class MainActivity extends AppCompatActivity{

    int appStatStopped = 0;  //the application is stopped
    int appStatStarted = 1;  //the application is started
    int appStatPaused = 2;   //the application is paused

    int appStat =  appStatStopped;  //by default

    Node node;
    private GameView gameView;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);

        setContentView(R.layout.activity_main);

        gameView = findViewById(R.id.canvas);

        Toolbar rr =  findViewById(R.id.toolbar_actionbar);
        rr.setSubtitle(rr.getSubtitle() + " on " + gameView.getNodeAddr() );
    }

    @Override
    public boolean onCreateOptionsMenu(Menu menu) {
        getMenuInflater().inflate(R.menu.menu_main, menu);
        return true;
    }

    private String getNodeAddr(String qrPeer){
        String nodeAddr = "";
        int index1 = qrPeer.indexOf("Babble*");
        if ( index1 > -1 ){
            index1 = index1 + 7;
            int index2 = qrPeer.indexOf("*", index1);
            if ( index2 > index1 ) {
                nodeAddr = qrPeer.substring(index1, index2);
            }
        }
        return nodeAddr;
    }

    private String getNodePublicKey(String qrPeer){
        String nodePublicKey = "";                 //Babble*NodeAddr*NodePublKey
        int index1 = qrPeer.indexOf("Babble*");
        if ( index1 > -1 ){
            index1 = index1 + 7;
            int index2 = qrPeer.indexOf("*", index1) + 1;
            if ( index2 > index1 ) {
                nodePublicKey = qrPeer.substring(index2);
            }
        }
        return nodePublicKey;
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {

        if ( requestCode == 9001 ) {  //RC_BARCODE_CAPTURE
            if ( resultCode == CommonStatusCodes.SUCCESS ) {
                if ( data != null ) {

                    ArrayList<String> newPeersDetected =  data.getStringArrayListExtra("newPeersDetected");   ////Babble*NodeAddr*NodePublicKey

                    for (int i =0; i < newPeersDetected.size(); i++ ){
                        gameView.addPeer(getNodeAddr(newPeersDetected.get(i)), getNodePublicKey(newPeersDetected.get(i)));   //add + save in configData
                    }
                } else {
                    Toast.makeText(getApplicationContext(), "onActivityResult: No QR code captured, intent data is null.", Toast.LENGTH_SHORT).show();
                }
            } else {
                Toast.makeText(getApplicationContext(), "onActivityResult: An unexpected error ocurred while reading QR code.", Toast.LENGTH_SHORT).show();
            }
        }else if ( requestCode == 9999 ){   //Save peers details into config file

            if ( data != null ) {
                 ArrayList<String> rawPeerData =  data.getStringArrayListExtra("rawPeerData");
                 if ( gameView != null ){
                     gameView.savePeerData(rawPeerData);
                     Toast.makeText(getApplicationContext(), "Successfully saved peers details.", Toast.LENGTH_SHORT).show();
                 }else{
                     Toast.makeText(getApplicationContext(), "onActivityResult: The peers details are not saved.", Toast.LENGTH_SHORT).show();
                 }
            }
        } else {
            super.onActivityResult(requestCode, resultCode, data);
        }
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {

        switch ( item.getItemId() ) {
            case R.id.start:
                if ( gameView.getPeersCount() < 2 ){
                    AlertDialog.Builder dlgAlert  = new AlertDialog.Builder(this);
                    dlgAlert.setMessage("You have empty peers collection.\r\nUse QR codes to collect more nodes.");
                    dlgAlert.setTitle("Babble.");
                    dlgAlert.setNeutralButton("OK", new DialogInterface.OnClickListener() {
                        public void onClick(DialogInterface dialog, int which) { }
                        });
                    dlgAlert.create().show();
                }else {
                    gameView.setBackgroundResource(0);

                    appStat = appStatStarted;
                    gameView.resume();
                }
                break;
            case R.id.pause:
                if ( gameView.getPeersCount() > 1 ) {
                    appStat = appStatPaused;
                    gameView.pause();
                }
                break;
            case R.id.scan:
                    ArrayList<String> allPeerSockets = gameView.loadAllPeerSockets();

                    Intent qrscanner = new Intent(this, BarcodeCaptureActivity.class);
                    qrscanner.putExtra("allPeerSockets", allPeerSockets);
                    startActivityForResult(qrscanner, 9001);  //RC_BARCODE_CAPTURE
                break;
            case R.id.generate:
                Intent qrgenerator = new Intent(this, ImageDialog.class);
                qrgenerator.putExtra("LocalNodeData", "Babble*" + gameView.getNodeAddr() + "*" + gameView.getNodePublicKey());
                startActivity(qrgenerator);
                break;
            case R.id.settings:
                ArrayList<String> peerData = gameView.loadRawPeerData();

                Intent nodeSettings = new Intent(this, ListViewDialog.class);
                nodeSettings.putStringArrayListExtra("RawNodeSettings", peerData );
                startActivityForResult(nodeSettings, 9999);
                break;
            case R.id.exit:
                finish();
                System.exit(0);
                break;
            default:
                return super.onOptionsItemSelected(item);
        }
        return true;
    }

    @Override
    protected void onPause() {
        if ( appStat != appStatStopped ) {
            Toast.makeText(getApplicationContext(), "Babble game paused.", Toast.LENGTH_SHORT).show();
            gameView.pause();
        }
        super.onPause();
    }

    @Override
    protected void onResume() {
        if ( appStat != appStatStopped ){
            Toast.makeText(getApplicationContext(), "Babble game started.", Toast.LENGTH_SHORT).show();
            gameView.resume();
        }
        super.onResume();
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        switch(keyCode){
            case KeyEvent.KEYCODE_BACK:
                //onBackPressed();          //to block this activity
                return true;
        }
        return super.onKeyDown(keyCode, event);
    }
}
