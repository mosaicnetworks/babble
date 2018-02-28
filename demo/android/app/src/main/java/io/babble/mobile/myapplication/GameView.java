package io.babble.mobile.myapplication;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.graphics.Path;
import android.util.AttributeSet;
import android.util.Log;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;

import java.util.HashMap;
import java.util.Map;

import mobile.Ball;

public class GameView extends SurfaceView implements Runnable {

    private final String TAG = "GameView";
    private Thread gameThread = null;

    private Context context;
    private SurfaceHolder surfaceHolder;
    private Paint paint;
    private boolean running;
    private float x, y;

    private BabbleNode node;

    public GameView(Context context) {
        this(context, null);
    }

    public GameView(Context context, AttributeSet attrs) {
        super(context, attrs);

        this.context = context;
        this.surfaceHolder = getHolder();

        paint = new Paint(Paint.ANTI_ALIAS_FLAG);
        paint.setColor(Color.WHITE);
        paint.setStyle(Paint.Style.FILL);

        node = new BabbleNode(context);
    }

    public void pause() {
        running = false;
        try {
            node.shutdown();
            gameThread.join();
        }
        catch (InterruptedException e) {
        }
    }

    public void resume() {
        running = true;

        node.run();
        gameThread = new Thread(this);
        gameThread.start();
    }

    @Override
    public boolean onTouchEvent(MotionEvent event) {
        float x = event.getX();
        float y = event.getY();

        Log.i("GameView", "onTouchEvent: " + x + "," + y);

        if (event.getAction() == MotionEvent.ACTION_DOWN) {
            this.x = x;
            this.y = y;
            invalidate();
        }

        return super.onTouchEvent(event);
    }

    @Override
    public void run() {
        Canvas canvas;

        while (running) {
            if (surfaceHolder.getSurface().isValid()) {
                canvas = surfaceHolder.lockCanvas();
                canvas.drawColor(Color.WHITE);

                for(Map.Entry<String, Ball> entry : node.store.entrySet()) {
                    String key = entry.getKey();
                    Ball value = entry.getValue();

                    paint.setColor(value.getColor());
                    canvas.drawCircle(value.getX(), value.getY(), value.getSize(), paint);

                    paint.setColor(Color.BLACK);
                    paint.setTextSize(36);
                    canvas.drawText(key, value.getX()-18, value.getY()+18, paint);
                }
                
                paint.setColor(Color.BLUE);
                canvas.drawCircle(x, y, 50, paint);

                paint.setColor(Color.WHITE);
                paint.setTextSize(36);
                canvas.drawText("L", x-18, y+18, paint);

                surfaceHolder.unlockCanvasAndPost(canvas);
            }
        }
    }
}
