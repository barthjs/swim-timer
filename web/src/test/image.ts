/** Creates the load event of an image element with the given natural size. */
export function imageLoad(width: number, height: number): Event {
  const image = document.createElement('img');
  Object.defineProperty(image, 'naturalWidth', {
    value: width,
  });
  Object.defineProperty(image, 'naturalHeight', {
    value: height,
  });
  const event = new Event('load');
  Object.defineProperty(event, 'target', {
    value: image,
  });
  return event;
}
