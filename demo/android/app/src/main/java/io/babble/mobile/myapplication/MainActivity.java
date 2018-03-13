package io.babble.mobile.myapplication;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.util.Log;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;

import mobile.ErrorHandler;
import mobile.Mobile;
import mobile.CommitHandler;
import mobile.EventHandler;
import mobile.Node;

public class MainActivity extends AppCompatActivity {

    Node node;
    private GameView gameView;
    private final String TAG = "App";

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
//        setContentView(R.layout.activity_main);

        gameView = new GameView(this);
        setContentView(gameView);



    }

    @Override
    protected void onPause() {
        super.onPause();
        Log.i(TAG, "onPause: Pause GameView");
        gameView.pause();
    }

    @Override
    protected void onResume() {
        super.onResume();
        Log.i(TAG, "onResume: Start GameView");
        gameView.resume();
    }
}
