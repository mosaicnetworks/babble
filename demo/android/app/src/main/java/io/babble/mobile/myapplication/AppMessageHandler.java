package io.babble.mobile.myapplication;

import android.app.Activity;
import android.content.Context;
import android.util.Log;
import android.widget.TextView;

import mobile.Message;
import mobile.MessageHandler;

public class AppMessageHandler implements MessageHandler {
    protected MainActivity context;

    public AppMessageHandler(Context context) {
        this.context = (MainActivity)context;
    }

    @Override
    public void onMessage(final Message message) {
        context.runOnUiThread(new Runnable() {
            @Override
            public void run() {
                Log.i("Babble", "Received OnMessage event");
                TextView tv = context.findViewById(R.id.text);
                tv.setText(message.getData());
            }
        });
    }
}
