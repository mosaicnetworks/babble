package io.babble.mobile.myapplication;

import android.content.Context;
import android.graphics.Typeface;
import android.util.TypedValue;
import android.view.KeyEvent;
import android.view.LayoutInflater;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.EditorInfo;
import android.widget.AdapterView;
import android.widget.ArrayAdapter;
import android.widget.CheckBox;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.ListView;
import android.widget.TextView;
import android.widget.Toast;

import java.util.ArrayList;

public class AdapterListView extends ArrayAdapter<ListViewItem> {
    int resource;
    Context mContext;
    LayoutInflater inflater;
    ArrayList<ListViewItem> arrayPeers;

    public AdapterListView(Context context, int resource, ArrayList<ListViewItem> arrayPeers) {
        super(context, resource, arrayPeers);
        this.resource = resource;
        this.arrayPeers = arrayPeers;
    }

    @Override
    public int getCount() {
        if(arrayPeers != null && arrayPeers.size() != 0){
            return arrayPeers.size();
        }
        return 0;
    }

    @Override
    public ListViewItem getItem(int position) {
        return arrayPeers.get(position);
    }


    @Override
    public long getItemId(int position) {
        return position;
    }

    @Override
    public View getView(int position, View convertView, ViewGroup parent) {
        LinearLayout contactsView;
        ListViewItem peer = getItem(position);

        if (convertView == null) {
            contactsView = new LinearLayout(getContext());
            String inflater = Context.LAYOUT_INFLATER_SERVICE;
            LayoutInflater vi;
            vi = (LayoutInflater) getContext().getSystemService(inflater);
            vi.inflate(resource, contactsView, true);
        } else {
            contactsView = (LinearLayout) convertView;
        }

        CheckBox cbActive = (CheckBox) contactsView.findViewById(R.id.cbActive);

        TextView nodeAddr = (TextView) contactsView.findViewById(R.id.nodeAddr);
        nodeAddr.setTypeface(null, Typeface.BOLD);
        nodeAddr.setTextSize(TypedValue.COMPLEX_UNIT_PX, 18);

        final TextView edNickName = (TextView) contactsView.findViewById(R.id.edNickName);
        edNickName.setTextSize(TypedValue.COMPLEX_UNIT_PX, 22);

        cbActive.setChecked(peer.getActive());
        cbActive.setTag(position);
        nodeAddr.setText(peer.getNodeAddr());
        nodeAddr.setTag(position);
        edNickName.setText(peer.getNickName());
        edNickName.setTag(position);

        edNickName.setOnFocusChangeListener(new View.OnFocusChangeListener() {
            @Override
            public void onFocusChange(View v, boolean hasFocus) {
                if (!hasFocus) {
                    int itemIndex = (Integer) v.getTag();
                    String enteredNickName = ((EditText) v).getText().toString();
                    arrayPeers.get(itemIndex).setNickName(enteredNickName);
                }
            }
        });

        edNickName.setOnEditorActionListener(new TextView.OnEditorActionListener()
        {
            @Override
            public boolean onEditorAction(TextView v, int actionId, KeyEvent event)
            {
                if(actionId == EditorInfo.IME_ACTION_DONE)
                {
                    int itemIndex = (Integer) v.getTag();
                    String enteredNickName = ((EditText) v).getText().toString();
                    arrayPeers.get(itemIndex).setNickName(enteredNickName);
                }
                return false; // pass on to other listeners.
            }
        });

        cbActive.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (v != null){
                    int itemIndex = (Integer) v.getTag();
                    Boolean cbActive = ((CheckBox) v).isChecked();
                    arrayPeers.get(itemIndex).setActive(cbActive) ;
                }
            }
        });

        return contactsView;
    }

}
