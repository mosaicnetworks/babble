package io.babble.mobile.myapplication;

import android.app.Activity;
import android.content.Context;
import android.util.Log;
import android.widget.TextView;

import mobile.CommitHandler;
import mobile.TxContext;

public class AppMessageHandler implements CommitHandler {
    protected MainActivity context;

    public AppMessageHandler(Context context) {
        this.context = (MainActivity)context;
    }

    @Override
    public void onCommit(final TxContext msg) {
        context.runOnUiThread(new Runnable() {
            @Override
            public void run() {
                Log.i("Babble", "Received OnMessage event");
                TextView tv = context.findViewById(R.id.text);
                tv.setText(msg.getData());
            }
        });
    }
}
