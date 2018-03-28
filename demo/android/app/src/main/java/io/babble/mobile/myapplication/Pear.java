package io.babble.mobile.myapplication;

import com.google.gson.annotations.SerializedName;
import java.io.Serializable;


public class Pear implements Serializable {

    @SerializedName("NetAddr")
    public String netAddr;

    @SerializedName("PubKeyHex")
    public String pubKeyHex;
}
