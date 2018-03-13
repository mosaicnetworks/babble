package io.babble.mobile.myapplication;

import android.content.Context;
import android.util.Log;
import android.widget.TextView;

import java.util.HashMap;
import java.util.Map;

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

    public BabbleNode(Context context) {
        this.context = (MainActivity) context;

        EventHandler events = Mobile.newEventHandler();
        events.onMessage(this);
        events.onError(this);

        String nodeAddr = "10.128.1.136:1337";

        String peers = "[\n" +
                "    {\n" +
                "            \"NetAddr\":\"10.128.1.26:1337\",\n" +
                "            \"PubKeyHex\":\"0x04DB3C17CCE6F68B6A78137B5A39D427D2E42A787AEC6EFA749446CB9E9E6EB4C1D486DE94CD5EEA2ABE21C54E59A31045A4BF389E2E9A585260B4BBDB3EDB4440\"\n" +
                "    },\n" +
                "\t{\n" +
                "            \"NetAddr\":\"" + nodeAddr + "\",\n" +
                "            \"PubKeyHex\":\"0x043DE6FB7ED40017FB04B72BD8B99BE4A75851D8EFD1D4B4A1EAB7A9C0285DA135525D44930A69DCD227FCA78640D7C2DCF9AB1C23BC3E5B880296F4B878923F6F\"\n" +
                "    }\n" +
                "]";

        String privKey = "-----BEGIN EC PRIVATE KEY-----\n" +
                "MHcCAQEEIGSIiW4O+PI34+8ZB9ZYQw3gpmxQuG6CdhfQLemoBNPvoAoGCCqGSM49\n" +
                "AwEHoUQDQgAEPeb7ftQAF/sEtyvYuZvkp1hR2O/R1LSh6repwChdoTVSXUSTCmnc\n" +
                "0if8p4ZA18Lc+ascI7w+W4gClvS4eJI/bw==\n" +
                "-----END EC PRIVATE KEY-----";

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
