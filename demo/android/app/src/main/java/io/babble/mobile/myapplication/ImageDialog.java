package io.babble.mobile.myapplication;

import android.app.Activity;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Canvas;
import android.graphics.Color;
import android.graphics.Matrix;
import android.os.Bundle;
import android.widget.ImageView;

import com.google.zxing.BarcodeFormat;
import com.google.zxing.EncodeHintType;
import com.google.zxing.MultiFormatWriter;
import com.google.zxing.WriterException;
import com.google.zxing.common.BitMatrix;
import com.google.zxing.qrcode.decoder.ErrorCorrectionLevel;

import java.util.HashMap;
import java.util.Map;

public class ImageDialog extends Activity{

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.image_layout);
        String str = getIntent().getExtras().getString("LocalNodeData");

        Bitmap bm = pear2QRBitmap(str);
        ImageView iv = findViewById(R.id.your_image);
        iv.setImageBitmap(bm);
    }

    private Bitmap mergeBitmaps(Bitmap overlay, Bitmap bitmap) {
        int height = bitmap.getHeight();
        int width = bitmap.getWidth();
        Bitmap combined = Bitmap.createBitmap(width, height, bitmap.getConfig());
        Canvas canvas = new Canvas(combined);
        int canvasWidth = canvas.getWidth();
        int canvasHeight = canvas.getHeight();
        canvas.drawBitmap(bitmap, new Matrix(), null);
        int centreX = (canvasWidth  - overlay.getWidth()) / 2;
        int centreY = (canvasHeight - overlay.getHeight()) / 2 ;
        canvas.drawBitmap(overlay, centreX, centreY, null);
        return combined;
    }

    private Bitmap pear2QRBitmap(String value) {

        MultiFormatWriter multiFormatWriter = new MultiFormatWriter();
        try {
            Map<EncodeHintType, ErrorCorrectionLevel> hintMap =new HashMap<EncodeHintType, ErrorCorrectionLevel>();
            hintMap.put(EncodeHintType.ERROR_CORRECTION, ErrorCorrectionLevel.H);
            BitMatrix bitMatrix = multiFormatWriter.encode(value, BarcodeFormat.QR_CODE, 390, 390, hintMap);

            //converting bitmatrix to bitmap
            int width = bitMatrix.getWidth();
            int height = bitMatrix.getHeight();
            int[] pixels = new int[width * height];

            // All are 0, or black, by default
            for (int y = 0; y < height; y++) {
                int offset = y * width;
                for (int x = 0; x < width; x++) {
                    pixels[offset + x] = bitMatrix.get(x, y) ? Color.BLACK : Color.WHITE;
                }
            }
            Bitmap bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888);
            bitmap.setPixels(pixels, 0, width, 0, 0, width, height);

            //setting logo as bitmap
            Bitmap overlay = BitmapFactory.decodeResource(getResources(), R.drawable.babble64x64);

            return  mergeBitmaps(overlay, bitmap);

        } catch (WriterException e) {
            e.printStackTrace();
        }
        return null;
    }
}