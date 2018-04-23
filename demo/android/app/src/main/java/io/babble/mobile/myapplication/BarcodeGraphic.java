/*
 * Copyright (C) The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package io.babble.mobile.myapplication;

import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Paint;
import android.graphics.RectF;

import io.babble.mobile.myapplication.GameView;

import io.babble.mobile.myapplication.ui.camera.GraphicOverlay;
import com.google.android.gms.vision.barcode.Barcode;

import java.util.ArrayList;

/**
 * Graphic instance for rendering barcode position, size, and ID within an associated graphic
 * overlay view.
 */
public class BarcodeGraphic extends GraphicOverlay.Graphic {

    private ArrayList<String> activePeerSockets;    //a simple array list of active sockets

    private int mId;

    private static int mCurrentColorIndex = 0;

    private Paint mRectPaint;
    private Paint mTextPaint;
    private volatile Barcode mBarcode;

    BarcodeGraphic(GraphicOverlay overlay, ArrayList<String> aps) {
        super(overlay);

        ArrayList<String> activePeerSockets = aps;  // GameView.loadActivePeerSockets();
        mRectPaint = new Paint();
        mRectPaint.setStyle(Paint.Style.STROKE);
        mRectPaint.setStrokeWidth(4.0f);

        mTextPaint = new Paint();
        mTextPaint.setTextSize(24.0f);
    }

    public int getId() {
        return mId;
    }

    public void setId(int id) {
        this.mId = id;
    }

    public Barcode getBarcode() {
        return mBarcode;
    }

    /**
     * Updates the barcode instance from the detection of the most recent frame.  Invalidates the
     * relevant portions of the overlay to trigger a redraw.
     */
    void updateItem(Barcode barcode) {
        mBarcode = barcode;
        postInvalidate();
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

    /**
     * Draws the barcode annotations for position, size, and raw value on the supplied canvas.
     */
    @Override
    public void draw(Canvas canvas) {
        Barcode barcode = mBarcode;
        if (barcode == null) {
            return;
        }

        // Draws the bounding box around the barcode.
        RectF rect = new RectF(barcode.getBoundingBox());
        rect.left = translateX(rect.left);
        rect.top = translateY(rect.top);
        rect.right = translateX(rect.right);
        rect.bottom = translateY(rect.bottom);

        // Draws a label at the bottom of the barcode indicate the barcode value that was detected.

        int selectedColor = Color.RED;
        String socket = getNodeAddr(barcode.displayValue);
        if ((barcode.displayValue.contains("Babble")) && (!activePeerSockets.contains(socket))){
            selectedColor = Color.GREEN;
        }

        mRectPaint.setColor(selectedColor);
        canvas.drawRect(rect, mRectPaint);
    }
}
