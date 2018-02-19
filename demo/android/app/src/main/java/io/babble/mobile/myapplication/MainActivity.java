package io.babble.mobile.myapplication;

import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;

import mobile.ErrorHandler;
import mobile.Mobile;
import mobile.MessageHandler;
import mobile.EventHandler;
import mobile.Node;

public class MainActivity extends AppCompatActivity implements Runnable {

    Node node;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        EventHandler events = Mobile.newEventHandler();
        MessageHandler messageHandler = new AppMessageHandler(this);

        events.onMessage(messageHandler);
        events.onError(null);

//        Peer[] peers = new Peer[2];
//
//        peers[0] = new Peer();
//        peers[0].setNetAddr("10.128.1.26:1337");
//        peers[0].setPublicKeyHex("0x04DD4DEEA8299D878DC89BE556156E97953000E75D00CA79DB10023028F685547A7DC9C743E4D0F72CBC5F22D7C20C7921004BBA46DE7017187592681153312ED4");
//
//        peers[1] = new Peer();
//        peers[1].setNetAddr("10.128.1.136:1337");
//        peers[1].setPublicKeyHex("0x040BF27F16C42951A40873627BEEAF1A15859F00D9289C9F469076B3606E633AF09FD448D5167A8474423F710BDCBF9B4043510936DD931AA7065B7478387E5835");

        String peers = "[\n" +
                "    {\n" +
                "            \"NetAddr\":\"10.128.1.26:1337\",\n" +
                "            \"PubKeyHex\":\"0x04DD4DEEA8299D878DC89BE556156E97953000E75D00CA79DB10023028F685547A7DC9C743E4D0F72CBC5F22D7C20C7921004BBA46DE7017187592681153312ED4\"\n" +
                "    },\n" +
                "\t{\n" +
                "            \"NetAddr\":\"10.128.1.136:1337\",\n" +
                "            \"PubKeyHex\":\"0x043C92595407FE2FD21981C11EFB30E57139A4442DA39E243B3B46E2CF55969B2B8B99C994B40BFF97D8D4B0B1308599F106120E6A3A16FB33F2D2F2EAF965C0CB\"\n" +
                "    }\n" +
                "]";

        String privKey = "-----BEGIN EC PRIVATE KEY-----\n" +
                "MHcCAQEEIClVNSHmZJCLvIlx9q4Df2mg/CkTrshXvBuQERG6voW/oAoGCCqGSM49\n" +
                "AwEHoUQDQgAEPJJZVAf+L9IZgcEe+zDlcTmkRC2jniQ7O0biz1WWmyuLmcmUtAv/\n" +
                "l9jUsLEwhZnxBhIOajoW+zPy0vLq+WXAyw==\n" +
                "-----END EC PRIVATE KEY-----";


        node = Mobile.new_(peers, privKey, events, Mobile.defaultConfig());
    }

    @Override
    public void run() {
        //node.run();
    }
}
