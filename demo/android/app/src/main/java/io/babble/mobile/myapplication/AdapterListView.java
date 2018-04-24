package io.babble.mobile.myapplication;

import android.app.Activity;
import android.content.Context;
import android.os.Bundle;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.widget.ArrayAdapter;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.ListView;
import android.widget.TextView;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

public class AdapterListView extends  Activity {  //ArrayAdapter<ListViewItem>,

    Context mContext;
    LayoutInflater inflater;
    ArrayList<ListViewItem> arrayPeers;

/*    public AdapterListView(Context context, int resource, ArrayList<ListViewItem> arrayPeers) {

        super(context, resource);

        this.mContext = context;
        this.arrayPeers = arrayPeers;
        this.inflater = (LayoutInflater) context.getSystemService(Context.LAYOUT_INFLATER_SERVICE);
    }*/

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_listview);

        final ArrayList<String> rawPeerData = getIntent().getStringArrayListExtra("data");
        ArrayList<ListViewItem> lvPeerData =  new ArrayList<ListViewItem>();

        for (int i = 0; i < rawPeerData.size(); i++){
            lvPeerData.add(new ListViewItem ("NodeAddr", "NilName", true));
        }

        final ListView listview = (ListView) findViewById(R.id.lvItems);
        final StableArrayAdapter adapter = new StableArrayAdapter(this, R.layout.item_listview, lvPeerData);
        listview.setAdapter(adapter);
    }

    //@Override
    public long getItemId(int position) {
        //String item = getItem(position);
        //return mIdMap.get(item);
        return 0;
    }

    private class StableArrayAdapter extends ArrayAdapter<ListViewItem> {

        HashMap<String, ListViewItem> mIdMap = new HashMap<String, ListViewItem>();

        public StableArrayAdapter(Context context, int textViewResourceId,
                                  List<ListViewItem> objects) {
            super(context, textViewResourceId, objects);
            for (int i = 0; i < objects.size(); ++i) {
                mIdMap.put(String.valueOf(i), objects.get(i));
            }
        }

        @Override
        public long getItemId(int position) {
            ListViewItem item = getItem(position);
            return Integer.valueOf(mIdMap.get(item).toString());
        }

        @Override
        public boolean hasStableIds() {
            return true;
        }

    }

    // @Override
    public View getView(final int position, View convertView, ViewGroup parent) {

        final ViewHolder holder;
        if (convertView == null) {
            convertView = inflater.inflate(R.layout.item_listview, null);
            holder = new ViewHolder();
            holder.tvPosition = (TextView) convertView.findViewById(R.id.nodeAddr);
            holder.edNickName = (EditText) convertView.findViewById(R.id.edNickName);
            holder.cbActive = (CheckBox) convertView.findViewById(R.id.cbActive);
            convertView.setTag(holder);
        } else {
            holder = (ViewHolder) convertView.getTag();
        }

        ListViewItem listViewItem = arrayPeers.get(position);
        holder.tvPosition.setText("1234");
        //holder.cbActive.setB

                //acti.setChecked(listViewItem.getActive());

        //Using setOnclickListener not setOnCheckedChangeListener
        holder.cbActive.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
            }
        });

//Fill EditText with the value you have in data source
        holder.edNickName.setText(listViewItem.getNickName());
        holder.edNickName.setId(position);

//we need to update adapter once we finish with editing
        holder.edNickName.setOnFocusChangeListener(new View.OnFocusChangeListener() {

            public void onFocusChange(View v, boolean hasFocus) {

                if (!hasFocus) {
                    final int position = v.getId();
                    final EditText Caption = (EditText) v;
                    arrayPeers.get(position).setNickName(Caption.getText().toString());
                }

            }

        });

        return convertView;
    }

    //@Override
    public int getCount() {
        return arrayPeers.size();
    }

    static class ViewHolder {
        TextView tvPosition;
        EditText edNickName;
        CheckBox cbActive;
    }

}
