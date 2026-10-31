// SPDX-License-Identifier: Unlicense OR MIT

@import UIKit;

#include <stdint.h>
#include "_cgo_export.h"
#include "framework_ios.h"

__attribute__ ((visibility ("hidden"))) Class gio_layerClass(void);

// Relic patch: GioView adopts UITextInput (not just UIKeyInput), so the
// keyboard gets the field's text and selection and can offer autocorrect,
// predictive text and dictation. Positions are UTF-16 offsets from the
// start of the field; ime_ios.go converts them for the editor.
@interface GioTextPosition : UITextPosition
@property (nonatomic) NSInteger offset;
+ (instancetype)at:(NSInteger)offset;
@end

@implementation GioTextPosition
+ (instancetype)at:(NSInteger)offset {
	GioTextPosition *p = [[GioTextPosition alloc] init];
	p.offset = MAX(offset, 0);
	return p;
}
@end

@interface GioTextRange : UITextRange
@property (nonatomic, strong) GioTextPosition *from;
@property (nonatomic, strong) GioTextPosition *to;
+ (instancetype)from:(NSInteger)a to:(NSInteger)b;
@end

@implementation GioTextRange
+ (instancetype)from:(NSInteger)a to:(NSInteger)b {
	GioTextRange *r = [[GioTextRange alloc] init];
	r.from = [GioTextPosition at:MIN(a, b)];
	r.to = [GioTextPosition at:MAX(a, b)];
	return r;
}
- (UITextPosition *)start { return self.from; }
- (UITextPosition *)end { return self.to; }
- (BOOL)isEmpty { return self.from.offset == self.to.offset; }
@end

@interface GioView: UIView <UITextInput>
@property uintptr_t handle;
@property (nonatomic, weak) id<UITextInputDelegate> inputDelegate;
@property (nonatomic, copy) NSDictionary<NSAttributedStringKey, id> *markedTextStyle;
@property (nonatomic, strong) UITextInputStringTokenizer *gioTokenizer;
@end

@implementation GioViewController

CGFloat _keyboardHeight;

- (void)loadView {
	gio_runMain();

	CGRect zeroFrame = CGRectMake(0, 0, 0, 0);
	self.view = [[UIView alloc] initWithFrame:zeroFrame];
	self.view.layoutMargins = UIEdgeInsetsMake(0, 0, 0, 0);
	UIView *drawView = [[GioView alloc] initWithFrame:zeroFrame];
	[self.view addSubview: drawView];
#if !TARGET_OS_TV
	drawView.multipleTouchEnabled = YES;
#endif
	drawView.preservesSuperviewLayoutMargins = YES;
	drawView.layoutMargins = UIEdgeInsetsMake(0, 0, 0, 0);
	onCreate((__bridge CFTypeRef)drawView, (__bridge CFTypeRef)self);
#if !TARGET_OS_TV
	[[NSNotificationCenter defaultCenter] addObserver:self
											 selector:@selector(keyboardWillChange:)
												 name:UIKeyboardWillShowNotification
											   object:nil];
	[[NSNotificationCenter defaultCenter] addObserver:self
											 selector:@selector(keyboardWillChange:)
												 name:UIKeyboardWillChangeFrameNotification
											   object:nil];
	[[NSNotificationCenter defaultCenter] addObserver:self
											 selector:@selector(keyboardWillHide:)
												 name:UIKeyboardWillHideNotification
											   object:nil];
#endif
	[[NSNotificationCenter defaultCenter] addObserver: self
											 selector: @selector(applicationDidEnterBackground:)
												 name: UIApplicationDidEnterBackgroundNotification
											   object: nil];
	[[NSNotificationCenter defaultCenter] addObserver: self
											 selector: @selector(applicationWillEnterForeground:)
												 name: UIApplicationWillEnterForegroundNotification
											   object: nil];
}

- (void)applicationWillEnterForeground:(UIApplication *)application {
	GioView *view = (GioView *)self.view.subviews[0];
	if (view != nil) {
		onStart(view.handle);
	}
}

- (void)applicationDidEnterBackground:(UIApplication *)application {
	GioView *view = (GioView *)self.view.subviews[0];
	if (view != nil) {
		onStop(view.handle);
	}
}

- (void)viewDidDisappear:(BOOL)animated {
	[super viewDidDisappear:animated];
	GioView *view = (GioView *)self.view.subviews[0];
	onDestroy(view.handle);
}

- (void)viewDidLayoutSubviews {
	[super viewDidLayoutSubviews];
	GioView *view = (GioView *)self.view.subviews[0];
	CGRect frame = self.view.bounds;
	// Adjust view bounds to make room for the keyboard.
	frame.size.height -= _keyboardHeight;
	view.frame = frame;
	gio_onDraw(view.handle);
}

- (void)didReceiveMemoryWarning {
	onLowMemory();
	[super didReceiveMemoryWarning];
}

#if !TARGET_OS_TV
- (void)keyboardWillChange:(NSNotification *)note {
	NSDictionary *userInfo = note.userInfo;
	CGRect f = [userInfo[UIKeyboardFrameEndUserInfoKey] CGRectValue];
	_keyboardHeight = f.size.height;
	[self.view setNeedsLayout];
}

- (void)keyboardWillHide:(NSNotification *)note {
	_keyboardHeight = 0.0;
	[self.view setNeedsLayout];
}
#endif
@end

static void handleTouches(int last, GioView *view, NSSet<UITouch *> *touches, UIEvent *event) {
	CGFloat scale = view.contentScaleFactor;
	NSUInteger i = 0;
	NSUInteger n = [touches count];
	for (UITouch *touch in touches) {
		CFTypeRef touchRef = (__bridge CFTypeRef)touch;
		i++;
		NSArray<UITouch *> *coalescedTouches = [event coalescedTouchesForTouch:touch];
		NSUInteger j = 0;
		NSUInteger m = [coalescedTouches count];
		for (UITouch *coalescedTouch in [event coalescedTouchesForTouch:touch]) {
			CGPoint loc = [coalescedTouch locationInView:view];
			j++;
			int lastTouch = last && i == n && j == m;
			onTouch(view.handle, lastTouch, touchRef, touch.phase, loc.x*scale, loc.y*scale, [coalescedTouch timestamp]);
		}
	}
}

@implementation GioView
NSArray<UIKeyCommand *> *_keyCommands;
+ (void)onFrameCallback:(CADisplayLink *)link {
       gio_onFrameCallback((__bridge CFTypeRef)link);
}
+ (Class)layerClass {
    return gio_layerClass();
}
- (void)willMoveToWindow:(UIWindow *)newWindow {
	self.contentScaleFactor = newWindow.screen.nativeScale;
    if (@available(iOS 13.0, *)) {
        [self registerSceneNotifications:newWindow];
    }else{
        [self registerWindowNotifications:newWindow];
    }
}

- (void)registerSceneNotifications:(UIWindow *)newWindow {
        if (self.window != nil) {
            [[NSNotificationCenter defaultCenter] removeObserver:self
                                                  name:UISceneDidActivateNotification
                                                  object:self.window.windowScene];
            [[NSNotificationCenter defaultCenter] removeObserver:self
                                                  name:UISceneWillDeactivateNotification
                                                  object:self.window.windowScene];
        }
        [[NSNotificationCenter defaultCenter] addObserver:self
                                              selector:@selector(onSceneDidActivate:)
                                              name:UISceneDidActivateNotification
                                              object:newWindow.windowScene];

        [[NSNotificationCenter defaultCenter] addObserver:self
                                              selector:@selector(onSceneWillDeactivate:)
                                              name:UISceneWillDeactivateNotification
                                              object:newWindow.windowScene];
}

- (void)onSceneDidActivate:(NSNotification *)note API_AVAILABLE(ios(13.0)){
    onFocus(self.handle, YES);
}

- (void)onSceneWillDeactivate:(NSNotification *)note API_AVAILABLE(ios(13.0)){
    onFocus(self.handle, NO);
}

- (void)registerWindowNotifications:(UIWindow *)newWindow {
	if (self.window != nil) {
		[[NSNotificationCenter defaultCenter] removeObserver:self
											  name:UIWindowDidBecomeKeyNotification
										      object:self.window];
		[[NSNotificationCenter defaultCenter] removeObserver:self
											  name:UIWindowDidResignKeyNotification
											  object:self.window];
	}
	[[NSNotificationCenter defaultCenter] addObserver:self
										  selector:@selector(onWindowDidBecomeKey:)
										  name:UIWindowDidBecomeKeyNotification
										  object:newWindow];
	[[NSNotificationCenter defaultCenter] addObserver:self
										  selector:@selector(onWindowDidResignKey:)
										  name:UIWindowDidResignKeyNotification
										  object:newWindow];
}

- (void)onWindowDidBecomeKey:(NSNotification *)note {
	if (self.isFirstResponder) {
		onFocus(self.handle, YES);
	}
}

- (void)onWindowDidResignKey:(NSNotification *)note {
	if (self.isFirstResponder) {
		onFocus(self.handle, NO);
	}
}

- (void)touchesBegan:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
	handleTouches(0, self, touches, event);
}

- (void)touchesMoved:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
	handleTouches(0, self, touches, event);
}

- (void)touchesEnded:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
	handleTouches(1, self, touches, event);
}

- (void)touchesCancelled:(NSSet<UITouch *> *)touches withEvent:(UIEvent *)event {
	handleTouches(1, self, touches, event);
}

// The field's text, selection and marked (composing) range, from Go.
- (NSString *)gioText:(NSRange *)sel marked:(NSRange *)marked {
	int s0, s1, c0, c1;
	NSString *text = (__bridge_transfer NSString *)gio_imeText(self.handle, &s0, &s1, &c0, &c1);
	if (sel != NULL) {
		*sel = NSMakeRange(s0, MAX(s1 - s0, 0));
	}
	if (marked != NULL) {
		*marked = c0 < 0 ? NSMakeRange(NSNotFound, 0) : NSMakeRange(c0, MAX(c1 - c0, 0));
	}
	return text == nil ? @"" : text;
}

- (NSInteger)gioClamp:(NSInteger)offset {
	NSInteger n = [self gioText:NULL marked:NULL].length;
	return MAX(0, MIN(offset, n));
}

- (void)insertText:(NSString *)text {
	NSRange marked;
	[self gioText:NULL marked:&marked];
	if (marked.location != NSNotFound) {
		// Committing a composition replaces the marked text.
		gio_imeReplace(self.handle, (int)marked.location, (int)NSMaxRange(marked), (__bridge CFTypeRef)text);
		NSInteger caret = marked.location + text.length;
		gio_imeSetComposing(self.handle, -1, -1);
		gio_imeSetSelection(self.handle, (int)caret, (int)caret);
		return;
	}
	onText(self.handle, (__bridge CFTypeRef)text);
}

- (NSString *)textInRange:(UITextRange *)range {
	GioTextRange *r = (GioTextRange *)range;
	NSString *text = [self gioText:NULL marked:NULL];
	NSInteger a = MIN(r.from.offset, (NSInteger)text.length), b = MIN(r.to.offset, (NSInteger)text.length);
	return [text substringWithRange:NSMakeRange(a, MAX(b - a, 0))];
}

- (void)replaceRange:(UITextRange *)range withText:(NSString *)text {
	GioTextRange *r = (GioTextRange *)range;
	gio_imeReplace(self.handle, (int)r.from.offset, (int)r.to.offset, (__bridge CFTypeRef)text);
}

- (UITextRange *)selectedTextRange {
	NSRange sel;
	[self gioText:&sel marked:NULL];
	return [GioTextRange from:sel.location to:NSMaxRange(sel)];
}

- (void)setSelectedTextRange:(UITextRange *)range {
	GioTextRange *r = (GioTextRange *)range;
	if (r == nil) {
		return;
	}
	gio_imeSetSelection(self.handle, (int)r.from.offset, (int)r.to.offset);
}

- (UITextRange *)markedTextRange {
	NSRange marked;
	[self gioText:NULL marked:&marked];
	if (marked.location == NSNotFound || marked.length == 0) {
		return nil;
	}
	return [GioTextRange from:marked.location to:NSMaxRange(marked)];
}

- (void)setMarkedText:(NSString *)markedText selectedRange:(NSRange)selectedRange {
	NSRange sel, marked;
	[self gioText:&sel marked:&marked];
	NSRange target = marked.location != NSNotFound ? marked : sel;
	if (markedText == nil) {
		markedText = @"";
	}
	gio_imeReplace(self.handle, (int)target.location, (int)NSMaxRange(target), (__bridge CFTypeRef)markedText);
	if (markedText.length > 0) {
		gio_imeSetComposing(self.handle, (int)target.location, (int)(target.location + markedText.length));
	} else {
		gio_imeSetComposing(self.handle, -1, -1);
	}
	NSInteger caret = target.location + MIN(selectedRange.location, markedText.length);
	gio_imeSetSelection(self.handle, (int)caret, (int)(caret + selectedRange.length));
}

- (void)unmarkText {
	gio_imeSetComposing(self.handle, -1, -1);
}

- (UITextPosition *)beginningOfDocument {
	return [GioTextPosition at:0];
}

- (UITextPosition *)endOfDocument {
	return [GioTextPosition at:[self gioText:NULL marked:NULL].length];
}

- (UITextRange *)textRangeFromPosition:(UITextPosition *)fromPosition toPosition:(UITextPosition *)toPosition {
	return [GioTextRange from:((GioTextPosition *)fromPosition).offset to:((GioTextPosition *)toPosition).offset];
}

- (UITextPosition *)positionFromPosition:(UITextPosition *)position offset:(NSInteger)offset {
	NSInteger p = ((GioTextPosition *)position).offset + offset;
	NSInteger n = [self gioText:NULL marked:NULL].length;
	if (p < 0 || p > n) {
		return nil;
	}
	return [GioTextPosition at:p];
}

- (UITextPosition *)positionFromPosition:(UITextPosition *)position inDirection:(UITextLayoutDirection)direction offset:(NSInteger)offset {
	switch (direction) {
	case UITextLayoutDirectionLeft:
	case UITextLayoutDirectionUp:
		return [self positionFromPosition:position offset:-offset];
	default:
		return [self positionFromPosition:position offset:offset];
	}
}

- (NSComparisonResult)comparePosition:(UITextPosition *)position toPosition:(UITextPosition *)other {
	NSInteger a = ((GioTextPosition *)position).offset, b = ((GioTextPosition *)other).offset;
	return a < b ? NSOrderedAscending : (a > b ? NSOrderedDescending : NSOrderedSame);
}

- (NSInteger)offsetFromPosition:(UITextPosition *)from toPosition:(UITextPosition *)toPosition {
	return ((GioTextPosition *)toPosition).offset - ((GioTextPosition *)from).offset;
}

- (id<UITextInputTokenizer>)tokenizer {
	if (self.gioTokenizer == nil) {
		self.gioTokenizer = [[UITextInputStringTokenizer alloc] initWithTextInput:self];
	}
	return self.gioTokenizer;
}

- (UITextPosition *)positionWithinRange:(UITextRange *)range farthestInDirection:(UITextLayoutDirection)direction {
	GioTextRange *r = (GioTextRange *)range;
	switch (direction) {
	case UITextLayoutDirectionLeft:
	case UITextLayoutDirectionUp:
		return r.from;
	default:
		return r.to;
	}
}

- (UITextRange *)characterRangeByExtendingPosition:(UITextPosition *)position inDirection:(UITextLayoutDirection)direction {
	NSInteger p = ((GioTextPosition *)position).offset;
	switch (direction) {
	case UITextLayoutDirectionLeft:
	case UITextLayoutDirectionUp:
		return [GioTextRange from:[self gioClamp:p - 1] to:p];
	default:
		return [GioTextRange from:p to:[self gioClamp:p + 1]];
	}
}

- (NSWritingDirection)baseWritingDirectionForPosition:(UITextPosition *)position inDirection:(UITextStorageDirection)direction {
	return NSWritingDirectionNatural;
}

- (void)setBaseWritingDirection:(NSWritingDirection)writingDirection forRange:(UITextRange *)range {
}

// The caret, from the editor (window pixels to view points).
- (CGRect)caretRectForPosition:(UITextPosition *)position {
	float x, y, h;
	gio_imeCaret(self.handle, &x, &y, &h);
	CGFloat s = self.contentScaleFactor;
	return CGRectMake(x / s, y / s, 2, MAX(h / s, 1));
}

- (CGRect)firstRectForRange:(UITextRange *)range {
	return [self caretRectForPosition:range.start];
}

- (NSArray<UITextSelectionRect *> *)selectionRectsForRange:(UITextRange *)range {
	return @[];
}

- (UITextPosition *)closestPositionToPoint:(CGPoint)point {
	NSRange sel;
	[self gioText:&sel marked:NULL];
	return [GioTextPosition at:NSMaxRange(sel)];
}

- (UITextPosition *)closestPositionToPoint:(CGPoint)point withinRange:(UITextRange *)range {
	return range.end;
}

- (UITextRange *)characterRangeAtPoint:(CGPoint)point {
	NSRange sel;
	[self gioText:&sel marked:NULL];
	return [GioTextRange from:sel.location to:NSMaxRange(sel)];
}

- (BOOL)canBecomeFirstResponder {
	return YES;
}

- (BOOL)hasText {
	return YES;
}

// Keyboard traits: sentences start with a capital, and autocorrect and
// spelling are on (the defaults for a text view).
- (UITextAutocapitalizationType)autocapitalizationType {
	return UITextAutocapitalizationTypeSentences;
}

- (UITextAutocorrectionType)autocorrectionType {
	return UITextAutocorrectionTypeYes;
}

- (UITextSpellCheckingType)spellCheckingType {
	return UITextSpellCheckingTypeYes;
}

- (void)deleteBackward {
	onDeleteBackward(self.handle);
}

- (void)onUpArrow {
	onUpArrow(self.handle);
}

- (void)onDownArrow {
	onDownArrow(self.handle);
}

- (void)onLeftArrow {
	onLeftArrow(self.handle);
}

- (void)onRightArrow {
	onRightArrow(self.handle);
}

- (NSArray<UIKeyCommand *> *)keyCommands {
	if (_keyCommands == nil) {
		_keyCommands = @[
			[UIKeyCommand keyCommandWithInput:UIKeyInputUpArrow
								modifierFlags:0
									   action:@selector(onUpArrow)],
			[UIKeyCommand keyCommandWithInput:UIKeyInputDownArrow
								modifierFlags:0
									   action:@selector(onDownArrow)],
			[UIKeyCommand keyCommandWithInput:UIKeyInputLeftArrow
								modifierFlags:0
									   action:@selector(onLeftArrow)],
			[UIKeyCommand keyCommandWithInput:UIKeyInputRightArrow
								modifierFlags:0
									   action:@selector(onRightArrow)]
		];
	}
	return _keyCommands;
}
@end

CFTypeRef gio_createDisplayLink(void) {
	CADisplayLink *dl = [CADisplayLink displayLinkWithTarget:[GioView class] selector:@selector(onFrameCallback:)];
	dl.paused = YES;
	NSRunLoop *runLoop = [NSRunLoop mainRunLoop];
	[dl addToRunLoop:runLoop forMode:[runLoop currentMode]];
	return (__bridge_retained CFTypeRef)dl;
}

int gio_startDisplayLink(CFTypeRef dlref) {
	CADisplayLink *dl = (__bridge CADisplayLink *)dlref;
	dl.paused = NO;
	return 0;
}

int gio_stopDisplayLink(CFTypeRef dlref) {
	CADisplayLink *dl = (__bridge CADisplayLink *)dlref;
	dl.paused = YES;
	return 0;
}

void gio_releaseDisplayLink(CFTypeRef dlref) {
	CADisplayLink *dl = (__bridge CADisplayLink *)dlref;
	[dl invalidate];
	CFRelease(dlref);
}

void gio_setDisplayLinkDisplay(CFTypeRef dl, uint64_t did) {
	// Nothing to do on iOS.
}

void gio_hideCursor() {
	// Not supported.
}

void gio_showCursor() {
	// Not supported.
}

void gio_setCursor(NSUInteger curID) {
	// Not supported.
}

void gio_viewSetHandle(CFTypeRef viewRef, uintptr_t handle) {
	GioView *v = (__bridge GioView *)viewRef;
	v.handle = handle;
}

@interface _gioAppDelegate : UIResponder <UIApplicationDelegate>
@property (strong, nonatomic) UIWindow *window;
@end

@implementation _gioAppDelegate
- (BOOL)application:(UIApplication *)application didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
	self.window = [[UIWindow alloc] initWithFrame:[[UIScreen mainScreen] bounds]];
	GioViewController *controller = [[GioViewController alloc] initWithNibName:nil bundle:nil];
	self.window.rootViewController = controller;
	[self.window makeKeyAndVisible];
	return YES;
}
- (BOOL)application:(UIApplication *)app openURL:(NSURL *)url options:(NSDictionary<UIApplicationOpenURLOptionsKey, id> *)options {
    gio_onOpenURI((__bridge CFTypeRef)url.absoluteString);
	return YES;
}
@end

int gio_applicationMain(int argc, char *argv[]) {
	@autoreleasepool {
		return UIApplicationMain(argc, argv, nil, NSStringFromClass([_gioAppDelegate class]));
	}
}

// Relic patch: tells the keyboard about changes it didn't make (another
// field, text set by the app, a tap moving the caret), so autocorrect
// works on the right words.
void gio_textInputChanged(CFTypeRef viewRef, int text, int selection) {
	GioView *view = (__bridge GioView *)viewRef;
	dispatch_async(dispatch_get_main_queue(), ^{
		id<UITextInputDelegate> d = view.inputDelegate;
		if (d == nil) {
			return;
		}
		if (text) {
			[d textWillChange:view];
			[d textDidChange:view];
		}
		if (selection) {
			[d selectionWillChange:view];
			[d selectionDidChange:view];
		}
	});
}
