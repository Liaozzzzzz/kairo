import { Languages } from '@/data/variables';
import { PlusOutlined } from '@ant-design/icons';
import { Button, Divider, Input, InputRef, Select, Space } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

type Props = {
  value: string | null;
  onChange: (value: string | null) => void;
};

export type SubtitlesLanguageSelectRef = {
  addLanguage: (language: string) => void;
};

const SubtitlesLanguageSelect = forwardRef<SubtitlesLanguageSelectRef, Props>(
  function SubtitlesLanguageSelect({ value, onChange }, ref) {
    const { t } = useTranslation();
    const inputRef = useRef<InputRef>(null);
    const [languages, setLanguages] = useState<string[]>([...Languages]);
    const [newLanguage, setNewLanguage] = useState('');

    const addItem = (e: React.MouseEvent<HTMLButtonElement | HTMLAnchorElement>) => {
      e.preventDefault();

      if (!newLanguage || languages.includes(newLanguage)) return;

      setLanguages([...languages, newLanguage]);
      onChange(newLanguage);
      setNewLanguage('');
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    };

    const addLanguage = (language: string) => {
      if (!language || languages.includes(language)) return;
      setLanguages([...languages, language]);
    };

    useImperativeHandle(ref, () => ({
      addLanguage,
    }));

    return (
      <Select
        className="w-full"
        placeholder={t('videos.subtitles.select.language_placeholder')}
        value={value}
        onChange={onChange}
        popupRender={(menu) => (
          <>
            {menu}
            <Divider style={{ margin: '8px 0' }} />
            <Space style={{ padding: '0 8px 4px' }}>
              <Input
                placeholder={t('videos.subtitles.select.language_custom_placeholder')}
                ref={inputRef}
                value={newLanguage}
                onChange={(e) => setNewLanguage(e.target.value)}
                onKeyDown={(e) => e.stopPropagation()}
              />
              <Button type="text" icon={<PlusOutlined />} onClick={addItem}>
                {t('common.add')}
              </Button>
            </Space>
          </>
        )}
        options={languages.map((item) => ({ label: item, value: item }))}
      />
    );
  }
);

export default SubtitlesLanguageSelect;
