ALTER TABLE personal_sites
ADD COLUMN search_indexing boolean NOT NULL DEFAULT false,
ADD CONSTRAINT personal_sites_search_indexing_requires_public
CHECK (NOT search_indexing OR visibility = 'public');
