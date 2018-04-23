package io.babble.mobile.myapplication;

import android.app.Dialog;
import android.graphics.Bitmap;
import android.view.View;
import android.view.Window;
import android.widget.Button;
import android.widget.ImageView;
import android.widget.Toast;

import com.google.zxing.BarcodeFormat;
import com.google.zxing.MultiFormatWriter;
import com.google.zxing.WriterException;
import com.google.zxing.common.BitMatrix;
import com.journeyapps.barcodescanner.BarcodeEncoder;

public class QRGenerate {

    private static MainActivity mainActivity;

/*    public QRGenerate(MainActivity mA, String qrData) {
        mainActivity = mA;
        //mainActivity.setContentView(R.layout.image_layout);
        genQR(qrData);
    }*/

    public static boolean genQR(MainActivity mA, String qrData){
        mainActivity = mA;
        Bitmap bm = pear2QRBitmap(qrData);
       // showImage(bm);
        return true;
    }

    private static Bitmap pear2QRBitmap(String value) {

        MultiFormatWriter multiFormatWriter = new MultiFormatWriter();
        try {
            BitMatrix bitMatrix = multiFormatWriter.encode(value, BarcodeFormat.QR_CODE, 200, 200);
            BarcodeEncoder barcodeEncoder = new BarcodeEncoder();
            Bitmap bitmap = barcodeEncoder.createBitmap(bitMatrix);
            return bitmap;
        } catch (WriterException e) {
            e.printStackTrace();
        }
        return null;
    }

   /* private static void showImage(Bitmap bm) {

       *//* Button buttonOne = (Button) mainActivity.findViewById(R.id.dismissListener);
        buttonOne.setOnClickListener(new Button.OnClickListener() {
            public void onClick(View v) {
                if (1==1){
                    Toast.makeText(mainActivity.getApplicationContext(), "qrgenerator: QR code not generated.", Toast.LENGTH_SHORT).show();
                }
            }
        });*//*

        Dialog settingsDialog = new Dialog(mainActivity);
        settingsDialog.getWindow().requestFeature(Window.FEATURE_NO_TITLE);

        ImageView iv = (ImageView)mainActivity.findViewById(R.id.imageQRCode);
        iv.setImageBitmap(bm);
        settingsDialog.setContentView(mainActivity.getLayoutInflater().inflate(R.layout.image_layout, null));
        settingsDialog.show();
    }*/





}
