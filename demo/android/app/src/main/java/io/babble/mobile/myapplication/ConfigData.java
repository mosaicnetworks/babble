package io.babble.mobile.myapplication;
import com.google.gson.annotations.SerializedName;

import java.io.Serializable;

public class ConfigData {

    @SerializedName("Peers")
    public Pear [] peers;

    @SerializedName("NodeID")
    public int nodeID;     //0 one of array indexes above

    @SerializedName("NodePrivateKey")
    public String nodePrivateKey;  //"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEEIP7PIEnr/7RUSLc55XP44GTsAxsSg/AzekqHqXDQxfGKoAoGCCqGSM49AwEHoUQDQgAEpM2p+b0DxYTEJaPtGiaM2oVjtixMbx5S6BrVUDREuH8A4rNKrAWohJFZHHG6U5w15y9KaTYntoB5Cq/mS0x6ww==\n-----END EC PRIVATE KEY-----",

    @SerializedName("NodePublicKey")
    public String nodePublicKey;   //"0x04A4CDA9F9BD03C584C425A3ED1A268CDA8563B62C4C6F1E52E81AD5503444B87F00E2B34AAC05A88491591C71BA539C35E72F4A693627B680790AAFE64B4C7AC3",

    @SerializedName("Node_addr")
    public String node_addr;       // "10.128.1.36:1331",

    @SerializedName("Proxy_addr")
    public String proxy_addr;      //"10.128.1.36:1332",

    @SerializedName("Client_addr")
    public String client_addr;     //"10.128.1.36:1333",

    @SerializedName("Service_addr")
    public String service_addr;    //"10.128.1.36:1334",

    @SerializedName("StoreType")
    public String storeType;       //"inmem" or "badger"

    @SerializedName("StorePath")
    public String storePath;        //"D:\\Projects\\go-work\\...

    @SerializedName("CircleBackColor")
    public String circleBackColor;  //"BLUE" alwyas upper case

    @SerializedName("CircleForeColor")
    public String circleForeColor;   //"WHITE" alwyas upper case

    @SerializedName("CircleRadius")
    public int circleRadius;         //[50, 150]

}
