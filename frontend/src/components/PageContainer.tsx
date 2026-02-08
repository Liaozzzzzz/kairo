import { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface PageContainerProps {
  children: ReactNode;
  className?: string;
  maxWidth?: string;
  header?: ReactNode;
  footer?: ReactNode;
  viewClass?: string;
}

const PageContainer = ({
  children,
  className,
  maxWidth = 'max-w-5xl',
  header,
  footer,
  viewClass,
}: PageContainerProps) => {
  const headerNode = header ? (
    <div className="flex justify-center w-full ">
      <div className={cn('w-full px-10', maxWidth)}>{header}</div>
    </div>
  ) : null;

  const footerNode = footer ? (
    <div className="flex justify-center w-full">
      <div className={cn('w-full px-10', maxWidth)}>{footer}</div>
    </div>
  ) : null;

  return (
    <div className="flex-1 w-full bg-background text-foreground flex flex-col">
      {headerNode && <div className="sticky top-0 z-50 bg-background pt-10">{headerNode}</div>}

      <div className="flex-1 w-full">
        <div className="flex justify-center w-full">
          <div className={cn('w-full py-6', maxWidth, className, viewClass)}>{children}</div>
        </div>
      </div>

      {footerNode && <div className="sticky bottom-0 z-50 bg-background pb-10">{footerNode}</div>}
    </div>
  );
};

export default PageContainer;
