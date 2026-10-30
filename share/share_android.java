// Android's share sheet for the Year card (share_android.go calls this).
// Compiled into share_android.jar, which gogio adds to the APK:
//
//	javac --release 8 -classpath android.jar -d classes share_android.java
//	jar cf share_android.jar -C classes .
//
// Any android.jar works (the methods used are old); the build only needs
// the class names.

package org.saniyaa61.relic.share;

import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.MediaStore;
import android.view.View;

import java.io.OutputStream;

public class share_android {
    // share puts png into Pictures/Relic (replacing an earlier copy with
    // the same name) and opens the share sheet for it. Other apps can only
    // read a picture the app hands them through the media store: the app
    // can't declare its own file provider. Returns an error message, or
    // null when the sheet is on its way.
    public static String share(final View view, byte[] png, String name, final String title) {
        if (Build.VERSION.SDK_INT < 29) {
            return "sharing needs Android 10 or newer";
        }
        final Context ctx = view.getContext();
        ContentResolver cr = ctx.getContentResolver();
        Uri images = MediaStore.Images.Media.EXTERNAL_CONTENT_URI;
        try {
            cr.delete(images, "_display_name = ? AND relative_path LIKE ?",
                    new String[]{name, "Pictures/Relic%"});
        } catch (Exception e) {
            // An old copy left behind is harmless.
        }
        ContentValues v = new ContentValues();
        v.put("_display_name", name);
        v.put("mime_type", "image/png");
        v.put("relative_path", "Pictures/Relic");
        v.put("is_pending", 1);
        final Uri uri;
        try {
            uri = cr.insert(images, v);
            if (uri == null) {
                return "the picture couldn't be stored";
            }
            OutputStream out = cr.openOutputStream(uri);
            try {
                out.write(png);
            } finally {
                out.close();
            }
            v.clear();
            v.put("is_pending", 0);
            cr.update(uri, v, null, null);
        } catch (Exception e) {
            return e.toString();
        }
        view.post(new Runnable() {
            public void run() {
                Intent send = new Intent(Intent.ACTION_SEND);
                send.setType("image/png");
                send.putExtra(Intent.EXTRA_STREAM, uri);
                send.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
                Intent chooser = Intent.createChooser(send, title);
                chooser.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
                try {
                    ctx.startActivity(chooser);
                } catch (Exception e) {
                    chooser.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                    ctx.startActivity(chooser);
                }
            }
        });
        return null;
    }
}
