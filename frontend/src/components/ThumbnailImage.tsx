import { useState, useEffect } from 'react';
import { Image, ImageProps } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { ReadImageBase64 } from '@root/wailsjs/go/main/App';
import { ImageFallback } from '@/data/variables';

interface ThumbnailImageProps extends ImageProps {
  src?: string;
}

export function ThumbnailImage({ src, fallback = ImageFallback, ...props }: ThumbnailImageProps) {
  const [imageSrc, setImageSrc] = useState<string | undefined>(src);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!src) {
      setImageSrc(undefined);
      return;
    }

    if (
      src.startsWith('http') ||
      src.startsWith('https') ||
      src.startsWith('blob:') ||
      src.startsWith('data:')
    ) {
      setImageSrc(src);
      return;
    }

    // Local path
    setLoading(true);
    ReadImageBase64(src)
      .then((base64: string) => {
        setImageSrc(base64);
      })
      .catch((err) => {
        console.error('Failed to load local thumbnail:', err);
        setImageSrc(undefined);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [src]);

  if (!imageSrc && !loading) {
    return (
      <div className="flex items-center justify-center w-full h-full text-gray-300 dark:text-muted-foreground/50 bg-gray-100 dark:bg-white/5">
        <PlayCircleOutlined className="w-4 h-4" />
      </div>
    );
  }

  return <Image src={imageSrc} fallback={fallback} {...props} />;
}
