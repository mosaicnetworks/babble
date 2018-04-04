package io.babble.mobile.myapplication;

import com.google.gson.annotations.SerializedName;
import java.io.Serializable;


public class Peer implements Serializable {

    @SerializedName("NetAddr")
    public String netAddr;

    @SerializedName("PubKeyHex")
    public String pubKeyHex;
}
