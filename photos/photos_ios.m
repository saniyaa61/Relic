//go:build ios

// The Photos picker (PHPickerViewController, iOS 14+) for photos_ios.go.
// The chosen picture is redrawn upright, at most 1600 px on its long
// edge, and handed to Go as JPEG (so HEIC photos work too).

#include <UIKit/UIKit.h>
#include <PhotosUI/PhotosUI.h>
#include "_cgo_export.h"

API_AVAILABLE(ios(14))
@interface RelicPhotoDelegate : NSObject <PHPickerViewControllerDelegate>
@end

static void deliver(UIImage *img) {
	if (img == nil) {
		relicPhotoPicked(NULL, 0, -1);
		return;
	}
	CGSize size = img.size;
	CGFloat longEdge = MAX(size.width, size.height);
	if (longEdge > 1600) {
		CGFloat k = 1600 / longEdge;
		size = CGSizeMake(round(size.width * k), round(size.height * k));
	}
	UIGraphicsImageRendererFormat *format = [UIGraphicsImageRendererFormat defaultFormat];
	format.scale = 1;
	format.opaque = YES;
	UIGraphicsImageRenderer *r = [[UIGraphicsImageRenderer alloc] initWithSize:size format:format];
	UIImage *upright = [r imageWithActions:^(UIGraphicsImageRendererContext *ctx) {
		[[UIColor whiteColor] setFill];
		UIRectFill(CGRectMake(0, 0, size.width, size.height));
		[img drawInRect:CGRectMake(0, 0, size.width, size.height)];
	}];
	NSData *jpg = UIImageJPEGRepresentation(upright, 0.9);
	if (jpg == nil) {
		relicPhotoPicked(NULL, 0, -1);
		return;
	}
	relicPhotoPicked((void *)jpg.bytes, (int)jpg.length, 1);
}

@implementation RelicPhotoDelegate
- (void)picker:(PHPickerViewController *)picker didFinishPicking:(NSArray<PHPickerResult *> *)results API_AVAILABLE(ios(14)) {
	[picker dismissViewControllerAnimated:YES completion:nil];
	if (results.count == 0) {
		relicPhotoPicked(NULL, 0, 0);
		return;
	}
	NSItemProvider *p = results.firstObject.itemProvider;
	if (![p canLoadObjectOfClass:[UIImage class]]) {
		relicPhotoPicked(NULL, 0, -1);
		return;
	}
	[p loadObjectOfClass:[UIImage class] completionHandler:^(id<NSItemProviderReading> obj, NSError *err) {
		deliver([obj isKindOfClass:[UIImage class]] ? (UIImage *)obj : nil);
	}];
}
@end

// The delegate must outlive the picker: PHPickerViewController holds it weakly.
static RelicPhotoDelegate *delegate;

int relicPickPhoto(CFTypeRef ctrlRef) {
	if (@available(iOS 14, *)) {
		UIViewController *ctrl = (__bridge UIViewController *)ctrlRef;
		while (ctrl.presentedViewController != nil) {
			ctrl = ctrl.presentedViewController;
		}
		PHPickerConfiguration *config = [[PHPickerConfiguration alloc] init];
		config.filter = [PHPickerFilter imagesFilter];
		config.selectionLimit = 1;
		PHPickerViewController *picker = [[PHPickerViewController alloc] initWithConfiguration:config];
		delegate = [[RelicPhotoDelegate alloc] init];
		picker.delegate = delegate;
		[ctrl presentViewController:picker animated:YES completion:nil];
		return 1;
	}
	return 0;
}
