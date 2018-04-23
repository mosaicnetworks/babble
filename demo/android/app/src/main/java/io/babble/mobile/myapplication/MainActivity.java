package io.babble.mobile.myapplication;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.support.v7.app.AppCompatActivity;
import android.support.v7.widget.Toolbar;

import android.view.KeyEvent;
import android.view.Menu;
import android.view.MenuItem;
import android.view.View;
import android.widget.AdapterView;

import android.widget.Toast;

import com.google.android.gms.common.api.CommonStatusCodes;
import com.google.android.gms.vision.barcode.Barcode;

import java.util.ArrayList;

import mobile.Node;

public class MainActivity extends AppCompatActivity{

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
        if (index1 > -1){
            index1 = index1 + 7;
            int index2 = qrPeer.indexOf("*", index1);
            if (index2 > index1) {
                nodeAddr = qrPeer.substring(index1, index2 - 1);
            }
        }
        return nodeAddr;
    }

    private String getNodePublicKey(String qrPeer){
        String nodePublicKey = "";
        int index1 = qrPeer.indexOf("Babble*");
        if (index1 > -1){
            index1 = index1 + 7;
            int index2 = qrPeer.indexOf("*", index1);
            if (index2 > index1) {
                nodePublicKey = qrPeer.substring(index1, index2 - 1);
            }
        }
        return nodePublicKey;
    }


    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {

        if (requestCode == 9001){  //RC_BARCODE_CAPTURE
            if (resultCode == CommonStatusCodes.SUCCESS) {
                if (data != null) {

                    Barcode barcode = data.getParcelableExtra(BarcodeCaptureActivity.BarcodeObject);
                    String result = barcode.displayValue;    //Babble*NodeAddr*NodePublicKey
                    gameView.addPeer(getNodeAddr(result), getNodePublicKey(result));   //add + save in configData

                } else {
                    Toast.makeText(getApplicationContext(), "No QR code captured, intent data is null.", Toast.LENGTH_SHORT).show();
                }
            } else {
                Toast.makeText(getApplicationContext(), "An unexpected error ocurred while reading QR code.", Toast.LENGTH_SHORT).show();
            }
        } else {
            super.onActivityResult(requestCode, resultCode, data);
        }
    }

    @Override
    public boolean onOptionsItemSelected(MenuItem item) {

        switch (item.getItemId()) {
            case R.id.qrscanner:
                ArrayList<String> activePeerSockets = gameView.loadActivePeerSockets();

                Intent qrscanner = new Intent(this, BarcodeCaptureActivity.class);
                qrscanner.putExtra("ActivePeerSockets", activePeerSockets);
                startActivityForResult(qrscanner, 9001);  //RC_BARCODE_CAPTURE
                break;
            case R.id.qrgenerator:
                Intent qrgenerator = new Intent(this, ImageDialog.class);
                qrgenerator.putExtra("LocalNodeData", "Babble*" + gameView.getNodeAddr() + "*" + gameView.getNodePublicKey());
                startActivity(qrgenerator);
                break;
            case R.id.nodemanage:
                Toast.makeText(getApplicationContext(), "Soon available.", Toast.LENGTH_SHORT).show();

                //Intent nodemanage = new Intent(this, AdapterListView.class);

                //ArrayList<String> peerData = gameView.loadPeerData();

                //nodemanage.putStringArrayListExtra("data", peerData );

                //Bundle bundle = new Bundle();    //ArrayList<io.babble.mobile.myapplication.Pear> loadPeerData
                //bundle.putParcelableArrayList("data", peerData);
                //nodemanage.putExtras(bundle);

                //startActivity(nodemanage);
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
        super.onPause();
        Toast.makeText(getApplicationContext(), "Babble game paused.", Toast.LENGTH_SHORT).show();
        gameView.pause();
    }

    @Override
    protected void onResume() {
        super.onResume();
        Toast.makeText(getApplicationContext(), "Babble game started.", Toast.LENGTH_SHORT).show();
        gameView.resume();
    }

    @Override
    public boolean onKeyDown(int keyCode, KeyEvent event) {
        switch(keyCode){
            case KeyEvent.KEYCODE_BACK:
                //onBackPressed();
                return true;
        }
        return super.onKeyDown(keyCode, event);
    }
}
