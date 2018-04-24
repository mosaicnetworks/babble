package io.babble.mobile.myapplication;

import android.content.Context;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.graphics.Rect;
import android.util.AttributeSet;
import android.util.Log;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;

import java.lang.reflect.Field;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
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

    QuadraticBezierData  getBezierPoint( Map<String, QuadraticBezierData> quadraticBezierData, String key, float newEndX, float neweEndY, float stepT){

        QuadraticBezierData itemBezier = quadraticBezierData.get(key);

        if (itemBezier == null) return null;

        if ((itemBezier.EndX != newEndX) || (itemBezier.EndY != neweEndY)){
            if (itemBezier.LastT > 1f) {
                itemBezier.LastT = 0f;
                itemBezier.StartX = itemBezier.CurrentX;
                itemBezier.StartY = itemBezier.CurrentY;
                itemBezier.EndX = newEndX;
                itemBezier.EndY = neweEndY;
                itemBezier.ControlX = (itemBezier.EndX - itemBezier.ControlX)  * 0.01f;
                itemBezier.ControlY = (itemBezier.EndY - itemBezier.ControlY)  * 0.01f;
             }else {
                itemBezier.ControlX = (newEndX - itemBezier.StartX) * 0.01f;
                itemBezier.ControlY = (neweEndY - itemBezier.StartY) * 0.01f;
                itemBezier.EndX = newEndX;
                itemBezier.EndY = neweEndY;
            }
        }

        if (itemBezier.LastT > (1.0f + stepT))                  //nothing to do
            return itemBezier;

        //Quadratic Bezier Curve
        //x[0]*mt*mt + 2*x[1]*mt*t + x[2]*t*t,
        //y[0]*mt*mt + 2*y[1]*mt*t + y[2]*t*t
        //https://stackoverflow.com/questions/26481342/how-to-calculate-quadratic-bezier-curve?utm_medium=organic&utm_source=google_rich_qa&utm_campaign=google_rich_qa
        float mt = 1.0f - itemBezier.LastT;
        itemBezier.CurrentX = itemBezier.StartX * mt * mt + 2 * itemBezier.ControlX * mt * itemBezier.LastT + itemBezier.EndX * itemBezier.LastT * itemBezier.LastT;
        itemBezier.CurrentY  = itemBezier.StartY * mt * mt + 2 * itemBezier.ControlY * mt * itemBezier.LastT + itemBezier.EndY * itemBezier.LastT * itemBezier.LastT;

        itemBezier.LastT += stepT;

        return itemBezier;
    }

    int getColorByName(String name) { 

        int colorId;

        try {
            Class color = Class.forName("android.graphics.Color");
            Field field = color.getField(name);
            colorId = field.getInt(null);

        } catch (Exception e) {
            e.printStackTrace();
            colorId = Color.RED;
        }

        return colorId;
    }

    String getIPByNodeID(String key) {

        String iP = "Undefined" ;
        try {

            int iD = Integer.parseInt(key.replace("N", ""));
            io.babble.mobile.myapplication.Peer[] arr = node.cnfgData.peers;

            Arrays.sort (arr, new Comparator<Peer>() {
                public int compare(Peer p1, Peer p2)
                {
                    return p1.pubKeyHex.compareTo(p2.pubKeyHex);
                }
            });
            iP = arr[iD].netAddr;
        } catch (Exception e) {
            e.printStackTrace();
        }

        return iP;
    }

    String getIPByPubKeyHex(String pubKeyHex) {

        String iP = "Undefined" ;

        try {
            Peer[] arr= node.cnfgData.peers;

            Arrays.sort (arr, new Comparator<Peer>() {
                public int compare(Peer p1, Peer p2)
                {
                    return p1.pubKeyHex.compareTo(p2.pubKeyHex);
                }
            });

        for (int i = 0; i < arr.length; i++) {
            if (arr[i].pubKeyHex.equals(pubKeyHex)){
                iP = arr[i].netAddr;
                break;
            }
        }
        } catch (Exception e) {
            e.printStackTrace();
        }

        return iP;
    }

    int getOptimalFontSize(String text, int maxWidth) {

        Paint  paint = new Paint();
        Rect bounds = new Rect();
        int bounds_width, optFontSize = 12;

        for (int i = optFontSize; i < 40; i++) {

            paint.setTextSize(i);
            paint.getTextBounds(text, 0, text.length(), bounds);
            bounds_width = bounds.width();

            if (bounds_width  < maxWidth){
                optFontSize = i;
            }else{
                break;
            }
        }

        return optFontSize;
    }

    void addNewNodes(Map<String, QuadraticBezierData> quadraticBezierData){

        String iP;

        QuadraticBezierData itemBezier = quadraticBezierData.get("Local");
        if (itemBezier == null) {  //Initialize the own current circle
            int cBackColor = getColorByName(node.cnfgData.circleBackColor);
            int cForeColor = getColorByName(node.cnfgData.circleForeColor);
            iP = getIPByPubKeyHex(node.cnfgData.nodePublicKey);
            int optimalFontSize = getOptimalFontSize(iP, 2 * node.cnfgData.circleRadius - 4);

            quadraticBezierData.put("Local", new QuadraticBezierData(x, y, x, y, x, y, x, y,
                    1f, cBackColor, cForeColor, node.cnfgData.circleRadius, iP, optimalFontSize));
        }

        for(Map.Entry<String, Ball> entry : node.store.entrySet()) {
            String key1 = entry.getKey();
            Ball value1 = entry.getValue();

            if(!quadraticBezierData.containsKey("B" + key1)){
                float xxx = (float)value1.getX();
                float yyy = (float)value1.getY();
                int cBackColor = value1.getCircleBackColor();
                int cForeColor = value1.getCircleForeColor();
                iP = getIPByNodeID(key1);

                int optimalFontSize = getOptimalFontSize(iP, 2 * value1.getSize() - 4);
                quadraticBezierData.put("B" + key1, new QuadraticBezierData(xxx, yyy, xxx, yyy, xxx, yyy, xxx, yyy,
                        1f, cBackColor, cForeColor, value1.getSize(), iP, optimalFontSize));
            }
        }
     }

    long timeRefreshNodes = System.currentTimeMillis();

    @Override
    public void run() {

        Canvas canvas;

        QuadraticBezierData itemBezier;
        Map<String, QuadraticBezierData> quadraticBezierData = new HashMap();
        addNewNodes(quadraticBezierData);

        while (running) {

            if ((System.currentTimeMillis() - timeRefreshNodes) > 5000 ) { //check for new nodes per 5 seconds
                timeRefreshNodes = System.currentTimeMillis();
                addNewNodes(quadraticBezierData);
            }

            if (surfaceHolder.getSurface().isValid()) {
                canvas = surfaceHolder.lockCanvas();
                canvas.drawColor(Color.WHITE);

                for(Map.Entry<String, Ball> entry : node.store.entrySet()) {

                    String key = entry.getKey();
                    Ball value = entry.getValue();

                    itemBezier = getBezierPoint(quadraticBezierData, "B" + key, (float)value.getX(), (float)value.getY(), 0.04f);
                    if (itemBezier == null) continue;

                    paint.setColor(Color.BLACK);               //Black color border
                    canvas.drawCircle(itemBezier.CurrentX, itemBezier.CurrentY, itemBezier.CircleRadius + 2, paint);

                    Log.i("WWWW", ">>>>>>>>>>>>>>>>>>>>>" + itemBezier.CircleBackColor + ", " + itemBezier.CircleForeColor);
                    paint.setColor(itemBezier.CircleBackColor);
                    canvas.drawCircle(itemBezier.CurrentX, itemBezier.CurrentY, itemBezier.CircleRadius, paint);

                    paint.setColor(itemBezier.CircleForeColor);
                    paint.setTextSize(itemBezier.OptimalFontSize);
                    canvas.drawText(itemBezier.IP, itemBezier.CurrentX - itemBezier.CircleRadius + 2, itemBezier.CurrentY + 1, paint);
                }

                itemBezier = getBezierPoint(quadraticBezierData, "Local", x, y, 0.04f);

                paint.setColor(Color.BLACK);               //Black color border
                canvas.drawCircle(itemBezier.CurrentX, itemBezier.CurrentY, itemBezier.CircleRadius + 2, paint);

                paint.setColor(itemBezier.CircleBackColor);
                canvas.drawCircle(itemBezier.CurrentX, itemBezier.CurrentY, itemBezier.CircleRadius, paint);

                paint.setColor(itemBezier.CircleForeColor);
                paint.setTextSize(itemBezier.OptimalFontSize);
                canvas.drawText(itemBezier.IP, itemBezier.CurrentX - itemBezier.CircleRadius + 2, itemBezier.CurrentY + 1, paint);

                surfaceHolder.unlockCanvasAndPost(canvas);
            }
        }

    }

    public String getNodeAddr(){
        if (node != null){
            return node.cnfgData.node_addr;
        }else {
            return "";
        }
    }

    public String getNodePublicKey(){
        if (node != null){
            return node.cnfgData.nodePublicKey;
        }else {
            return "";
        }
    }

    public ConfigData getConfigData123(){
        if (node != null){
            return node.cnfgData;
        }else {
            return null;
        }
    }

    public void addPeer(String nodeAddr, String nodePublicKey){
       if ((node != null) && (node.cnfgData != null)){
           try {
               node.cnfgData.peers = Arrays.copyOf(node.cnfgData.peers, node.cnfgData.peers.length + 1);
               node.cnfgData.peers[node.cnfgData.peers.length - 1].active = 1;
               node.cnfgData.peers[node.cnfgData.peers.length - 1].netAddr = nodeAddr;
               node.cnfgData.peers[node.cnfgData.peers.length - 1].nickName = "";
               node.cnfgData.peers[node.cnfgData.peers.length - 1].pubKeyHex = nodePublicKey;

               node.saveConfigData(node.cnfgData);
           } catch(Exception e){
          }
       }
    }

    public  ArrayList<String> loadActivePeerSockets(){

        ArrayList<String> activePeerSockets = new ArrayList();

        ConfigData cnfgD =  node.cnfgData;
        if (cnfgD != null) {
            for (int i = 0; i < cnfgD.peers.length; i++)
                if (cnfgD.peers[i].active == 1) {
                    activePeerSockets.add(cnfgD.peers[i].netAddr);
                }
        }

        return null;
    }

    public  ArrayList<String> loadPeerData(){ //each string is netAddr*pubKeyHex*active

        ArrayList<String> peerData = new ArrayList<String>();

        if ((node != null) &&( node.cnfgData != null)) {
            for (int i = 0; i < node.cnfgData.peers.length; i++) {
                peerData.add(node.cnfgData.peers[i].netAddr + "*" + node.cnfgData.peers[i].nickName + "*" + node.cnfgData.peers[i].active);
            }
        }

        return peerData;
    }

}
