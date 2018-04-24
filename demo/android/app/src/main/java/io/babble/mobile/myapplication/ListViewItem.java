package io.babble.mobile.myapplication;

public class ListViewItem {

    String nodeAddr;
    String nickName;
    boolean active;

    public ListViewItem(String nodeAddr, String nickName, boolean active) {
        this.nodeAddr = nodeAddr;
        this.nickName = nickName;
        this.active  = active;
    }

    public String getNodeAddr() {
        return this.nodeAddr;
    }

    public void setNodeAddr(String nodeAddr) {
        this.nodeAddr = nodeAddr;
    }

    public String getNickName() {
        return this.nickName;
    }

    public void setNickName(String nickName) {
        this.nickName = nickName;
    }

    public Boolean getActive() {
        return this.active;
    }

    public void setActive(Boolean active) {
        this.active = active;
    }
}
