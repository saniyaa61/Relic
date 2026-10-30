// Android's photo picker for posters (photos_android.go calls this).
// Compiled into photos_android.jar, which gogio adds to the APK:
//
//	javac --release 8 -classpath android.jar -d classes photos_android.java
//	jar cf photos_android.jar -C classes .
//
// Any android.jar works (newer constants are spelled out as strings).

package org.saniyaa61.relic.photos;

import android.app.Activity;
import android.app.Fragment;
import android.app.FragmentManager;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.net.Uri;
import android.os.Build;
import android.provider.MediaStore;
import android.view.View;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;

public class photos_android {
    // onPicked hands the picture to Go: data and err both null means the
    // user backed out.
    static native void onPicked(byte[] data, String err);

    static final int REQUEST = 0x5e1;
    static final String TAG = "relic_photos_fragment";

    // picker_fragment receives the picker's result for the activity.
    public static class picker_fragment extends Fragment {
        @Override
        public void onActivityResult(int request, int result, final Intent data) {
            super.onActivityResult(request, result, data);
            if (request != REQUEST) {
                return;
            }
            if (result != Activity.RESULT_OK || data == null || data.getData() == null) {
                onPicked(null, null);
                return;
            }
            final Activity activity = getActivity();
            new Thread(new Runnable() {
                public void run() {
                    try {
                        onPicked(read(activity, data.getData()), null);
                    } catch (Throwable e) {
                        onPicked(null, e.toString());
                    }
                }
            }).start();
        }
    }

    // pick opens the system photo picker (Android 13+) or the gallery.
    public static void pick(View view) {
        final Activity activity = (Activity) view.getContext();
        activity.runOnUiThread(new Runnable() {
            public void run() {
                try {
                    FragmentManager fm = activity.getFragmentManager();
                    Fragment f = fm.findFragmentByTag(TAG);
                    if (f == null) {
                        f = new picker_fragment();
                        fm.beginTransaction().add(f, TAG).commit();
                        fm.executePendingTransactions();
                    }
                    Intent i;
                    if (Build.VERSION.SDK_INT >= 33) {
                        i = new Intent("android.provider.action.PICK_IMAGES");
                        i.setType("image/*");
                    } else {
                        i = new Intent(Intent.ACTION_PICK);
                        i.setDataAndType(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, "image/*");
                    }
                    try {
                        f.startActivityForResult(i, REQUEST);
                    } catch (ActivityNotFoundException e) {
                        Intent any = new Intent(Intent.ACTION_GET_CONTENT);
                        any.setType("image/*");
                        any.addCategory(Intent.CATEGORY_OPENABLE);
                        f.startActivityForResult(any, REQUEST);
                    }
                } catch (Throwable e) {
                    onPicked(null, e.toString());
                }
            }
        });
    }

    // read returns the picture's bytes: as they are for formats Go reads
    // (JPEG, PNG, GIF, WebP), otherwise (HEIC and the like) decoded here
    // and re-encoded as JPEG.
    static byte[] read(Activity activity, Uri uri) throws Exception {
        InputStream in = activity.getContentResolver().openInputStream(uri);
        if (in == null) {
            throw new Exception("the picture couldn't be opened");
        }
        byte[] raw;
        try {
            ByteArrayOutputStream buf = new ByteArrayOutputStream();
            byte[] chunk = new byte[1 << 16];
            int n;
            while ((n = in.read(chunk)) > 0) {
                buf.write(chunk, 0, n);
                if (buf.size() > (60 << 20)) {
                    throw new Exception("the picture is too large");
                }
            }
            raw = buf.toByteArray();
        } finally {
            in.close();
        }
        if (goReads(raw)) {
            return raw;
        }
        BitmapFactory.Options o = new BitmapFactory.Options();
        o.inJustDecodeBounds = true;
        BitmapFactory.decodeByteArray(raw, 0, raw.length, o);
        int sample = 1;
        while (Math.max(o.outWidth, o.outHeight) / (sample * 2) >= 1200) {
            sample *= 2;
        }
        o = new BitmapFactory.Options();
        o.inSampleSize = sample;
        Bitmap bmp = BitmapFactory.decodeByteArray(raw, 0, raw.length, o);
        if (bmp == null) {
            throw new Exception("this kind of picture can't be read");
        }
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        bmp.compress(Bitmap.CompressFormat.JPEG, 92, out);
        bmp.recycle();
        return out.toByteArray();
    }

    static boolean goReads(byte[] b) {
        if (b.length < 12) {
            return false;
        }
        boolean jpeg = (b[0] & 0xff) == 0xff && (b[1] & 0xff) == 0xd8;
        boolean png = (b[0] & 0xff) == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G';
        boolean gif = b[0] == 'G' && b[1] == 'I' && b[2] == 'F';
        boolean webp = b[0] == 'R' && b[1] == 'I' && b[2] == 'F' && b[3] == 'F' && b[8] == 'W' && b[9] == 'E' && b[10] == 'B' && b[11] == 'P';
        return jpeg || png || gif || webp;
    }
}
