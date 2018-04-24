package io.babble.mobile.myapplication;

import com.google.gson.annotations.SerializedName;
import java.io.Serializable;


public class Peer implements Serializable {

    @SerializedName("NetAddr")
    public String netAddr;

    @SerializedName("PubKeyHex")
    public String pubKeyHex;

    @SerializedName("Active")
    public byte active;          //0,1

    @SerializedName("NickName")
    public String nickName;
}
