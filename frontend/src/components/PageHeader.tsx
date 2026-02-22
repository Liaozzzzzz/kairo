import { Typography } from 'antd';

const { Title, Text } = Typography;

const PageHeader = ({
  title,
  subtitle,
  extra,
}: {
  title: string;
  subtitle?: string;
  extra?: React.ReactNode;
}) => (
  <div className="flex items-center justify-between">
    <div className="flex items-center gap-8">
      <div>
        <Title className="!mb-0" level={2}>
          {title}
        </Title>
        {!!subtitle && (
          <Text className="mt-1" type="secondary">
            {subtitle}
          </Text>
        )}
      </div>
    </div>
    {extra}
  </div>
);

export default PageHeader;
