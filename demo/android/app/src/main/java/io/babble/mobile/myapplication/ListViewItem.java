package io.babble.mobile.myapplication;

public class ListViewItem {

    boolean active;
    String nickName;

    public ListViewItem(String nickName, boolean active) {
        this.nickName = nickName;
        this.active  = active;
    }

    public Boolean getActive() {
        return this.active;
    }

    public void setActive(Boolean active) {
        this.active = active;
    }

    public String getNickName() {
        return this.nickName;
    }

    public void setNickName(String nickName) {
        this.nickName = nickName;
    }
}
